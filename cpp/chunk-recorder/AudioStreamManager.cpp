/*
 * Copyright (C) 2018 Pascal Huerst <pascal.huerst@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

#include "AudioStreamManager.h"
#include "params.h"
#include "types.h"
#include "ServiceTracker.h"
#include "httplib.h"


#include <sys/ioctl.h>
#include <signal.h>
#include <stdint.h>
#include <sys/time.h>
#include <poll.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/stat.h>


AudioStreamManager::AudioStreamManager(const po::variables_map &vmCombined,
                                       CallbackDetector onDetectorStateChangedCB,
                                       CallbackStorage onStorageChangedCB) :
    m_terminateRequest(true),
    m_writeRawPcm(false),
    m_vmCombined(vmCombined),
    m_streamBuffer(nullptr),
    m_detectorBuffer(nullptr),
    m_storageBuffer(nullptr),
    m_alsaAudioInput(nullptr),
    m_streamWorker(nullptr),
    m_detectorWorker(nullptr),
    m_storageWorker(nullptr),
    m_detectorBufferSize(0),
    m_detectorSuccession(0),
    m_detectorThreshold(0.0),
    m_streamFifo(""),
    m_storageOutDir(""),
    m_streamPcmOutPrefix(""),
    m_streamStorageChunkSize(0),
    m_detectorStateChangedCB(onDetectorStateChangedCB),
    m_storageChangedCB(onStorageChangedCB)
{
}

AudioStreamManager::~AudioStreamManager()
{
}


bool sendBuffer(const std::string &url, const std::string &filename, SampleFrame *frames, size_t numFrames) {

    auto buffer = reinterpret_cast<const char*>(frames);
    size_t byteSize = (sizeof(SampleFrame) * numFrames); 

    std::string stringBuffer;
    for (int i=0; i<byteSize; i++) {
        stringBuffer += buffer[i];
    }    

    httplib::MultipartFormDataItems items = {
        {"raw_audio", stringBuffer, filename, "application/octet-stream" },
    };

    httplib::Client cli(url.c_str());
    auto res = cli.Post("/upload", items);

    if (res.error() != httplib::Error::Success || res->status != 200) {
        return false;
    }

    std::cout << "Successfully sent buffer: " <<  url << std::endl;

    return true;
}



void AudioStreamManager::start()
{
    if (m_terminateRequest) {

        m_terminateRequest = false;

        if (m_vmCombined.count(strOptStreamManagerFifo)) {
            m_streamFifo = m_vmCombined[strOptStreamManagerFifo].as<std::string>();

            if (!isValidPath(m_streamFifo)) {
                std::cout << "Fifo: " + m_streamFifo + " does not exist. Creating." << std::endl;
                createFifo(m_streamFifo);
                std::cout << "Pipe created at: " << m_streamFifo << " with size: " << static_cast<float>(fifoSize(File(m_streamFifo.c_str(), O_RDWR)) / 1024.0) << "kb" << std::endl;
            } else {
                if (!isFifo(File(m_streamFifo.c_str(), O_RDWR)))
                    throw std::invalid_argument(m_streamFifo + " is not a fifo.");
            }
        } else {
            throw std::invalid_argument(strOptStreamManagerFifo + " must be set!");
        }

        if (m_vmCombined.count(strOptStreamManagerStorageOutputDir)) {
            m_storageOutDir = m_vmCombined[strOptStreamManagerStorageOutputDir].as<std::string>();
        } else {
            throw std::invalid_argument(strOptStreamManagerStorageOutputDir + " must be set!");
        }

        if (m_vmCombined.count(strOptStreamManagerStoragePrefix)) {
            m_streamPcmOutPrefix = m_vmCombined[strOptStreamManagerStoragePrefix].as<std::string>();
        } else {
            throw std::invalid_argument(strOptStreamManagerStoragePrefix + " must be set!");
        }

        if (m_vmCombined.count(strOptStreamManagerPcmOutChunkSize)) {
            m_streamStorageChunkSize = m_vmCombined[strOptStreamManagerPcmOutChunkSize].as<unsigned long>();
        } else {
            throw std::invalid_argument(strOptStreamManagerPcmOutChunkSize + " must be set!");
        }

        m_storageBuffer.reset(new BlockingReaderWriterQueue<SampleFrame>(m_streamStorageChunkSize));

        unsigned int streamBufferSize = 0;
        if (m_vmCombined.count(strOptStreamManagerStreamBufferSize)) {
            streamBufferSize = m_vmCombined[strOptStreamManagerStreamBufferSize].as<unsigned int>();
        } else {
            throw std::invalid_argument(strOptStreamManagerStreamBufferSize + "must be set!");
        }

        m_streamBuffer.reset(new BlockingReaderWriterQueue<SampleFrame>(streamBufferSize));

        double detectorTotalTime = 0.0;
        if (m_vmCombined.count(strOptDetectorTotalTime)) {
            detectorTotalTime = m_vmCombined[strOptDetectorTotalTime].as<double>();
        } else {
            throw std::invalid_argument(strOptDetectorTotalTime + "must be set!");
        }

        double detectorWindowTime = 0.0;
        if (m_vmCombined.count(strOptDetectorWindowTime)) {
            detectorWindowTime = m_vmCombined[strOptDetectorWindowTime].as<double>();
        } else {
            throw std::invalid_argument(strOptDetectorWindowTime + "must be set!");
        }

        if (m_vmCombined.count(strOptDetectorThreshold)) {
            m_detectorThreshold = m_vmCombined[strOptDetectorThreshold].as<double>();
        } else {
            throw std::invalid_argument(strOptDetectorThreshold + "must be set!");
        }

        unsigned int rate = 0;
        if (m_vmCombined.count(strOptAudioRate)) {
            rate = m_vmCombined[strOptAudioRate].as<unsigned int>();
        } else {
            throw std::invalid_argument(strOptAudioRate + "must be set!");
        }

        m_detectorBufferSize = static_cast<size_t>(static_cast<double>(rate) * detectorWindowTime);
        m_detectorSuccession = static_cast<unsigned int>(detectorTotalTime / detectorWindowTime);
        m_detectorBuffer.reset(new BlockingReaderWriterQueue<SampleFrame>(m_detectorBufferSize));


        // Instatiation and lambda callback from alsa. Just fill out ringbuffers
        m_alsaAudioInput.reset(new AlsaAudioInput(m_vmCombined, [&] (SampleFrame *frames, size_t numFrames) {
                                   for (size_t i=0; i<numFrames; ++i) {
                                       m_streamBuffer->enqueue(frames[i]);
                                       m_detectorBuffer->enqueue(frames[i]);
                                       m_storageBuffer->enqueue(frames[i]);
                                   }
                               }));
#if 0
        // Stream lamda in a thread
        m_streamWorker.reset(new std::thread([&] {
            // Ignore SIGPIPE. Otherwise we terminate if reader disappears. Instead, we want start from the beginning
            ::signal(SIGPIPE, SIG_IGN);

            const size_t chunkSize = 1024;
            SampleFrame buffer[chunkSize];

            int counter = 0;
            while (!fifoHasReader(m_streamFifo) && !m_terminateRequest) {
                std::this_thread::sleep_for(std::chrono::milliseconds(500));
                if ((++counter) % 10 == 0) {
                    std::cout << "Pipe has no reader. (" << (counter>>1) << "s)" << std::endl;
                }
            }

            if (!m_terminateRequest) {
                // Now we had a reader. Open blocking.
                File fifoFd(m_streamFifo, O_WRONLY);

                while (!m_terminateRequest) {

                    size_t i=0;
                    while (i<chunkSize && !m_terminateRequest) {
                        if(!m_streamBuffer->wait_dequeue_timed(buffer[i++], std::chrono::milliseconds(500)))
                            continue;
                    }

                    if (m_terminateRequest)
                        continue;

                    size_t byteSize = chunkSize * sizeof(SampleFrame);
                    size_t bytesWritten = 0;

                    uint8_t *byteBuffer = reinterpret_cast<uint8_t*>(buffer);

                    while (bytesWritten < byteSize && !m_terminateRequest) {
                        try {
                            struct pollfd pfd = {0, 0, 0};
                            pfd.fd = fifoFd;
                            pfd.events = POLLOUT;

                            int ret = poll(&pfd, 1, 500);

                            if (ret == 0) { //Timeout
                                int availableBytes = fifoBytesAvailable(fifoFd);
                                size_t size = fifoSize(fifoFd);

                                std::cout << "Writing to Fifo timed out:\n";
                                std::cout << "available:  " << availableBytes << "\n";
                                std::cout << "size:       " << size << "\n";
                                std::cout << "Level:      " << (static_cast<float>(availableBytes) / static_cast<float>(availableBytes)) << "\n";

                                continue;

                            } else if (ret < 0) {
                                throw std::runtime_error(std::string("Error in poll(): ") + strerror(errno));

                            } // ret>0 -> Some data can be written

                            ssize_t written = 0;
                            while ((written = ::write(fifoFd, &byteBuffer[bytesWritten], byteSize - bytesWritten)) < 0 && errno == EPIPE && !m_terminateRequest) {
                                std::cout << "Fifo lost reader! write throws SIGPIPE. Waiting 500ms." << std::endl;
                                std::this_thread::sleep_for(std::chrono::milliseconds(500));
                            }

                            if (m_terminateRequest)
                                continue;

                            if (written < 0)
                                throw std::runtime_error(std::string("Error writing to fifo: (") + m_streamFifo + ") - " + strerror(errno));

                            bytesWritten += static_cast<size_t>(written);
                        
                        } catch (std::exception &e) {
                            std::cerr << e.what() << std::endl;
                            continue;
                        }
                    }
                }
            }
        }));
#endif
        // Detector lambda in a thread
        m_detectorWorker.reset(new std::thread([&] {

            SampleFrame buffer[m_detectorBufferSize];
            DetectorState currentState = {0.0, STATE_SILENT};
            unsigned int rmsCounter = 0;

            while (!m_terminateRequest) {

                size_t i=0;
                while (i<m_detectorBufferSize && !m_terminateRequest) {
                    auto ret = m_detectorBuffer->wait_dequeue_timed(buffer[i], std::chrono::milliseconds(500));
                    if (m_terminateRequest) return;
                    if (!ret)
                        continue;

                    i++;
                }

                double chunkSum = 0.0;
                for (unsigned int i=0; i<m_detectorBufferSize; ++i) {

                    double monoSum = (buffer[i].left + buffer[i].right) / 2.0;
                    chunkSum += (monoSum * monoSum);
                }

                double chunkSumMean = static_cast<double>(chunkSum) / static_cast<double>(m_detectorBufferSize);
                double rms = sqrt(chunkSumMean);
                currentState.rmsPercent = 100.0 * rms / static_cast<double>(std::numeric_limits<Sample>::max());

                if (currentState.rmsPercent < m_detectorThreshold) {
                    if (rmsCounter > 0) {
                        rmsCounter--;
                        if (rmsCounter == 0 && currentState.state == STATE_SIGNAL) {
                            currentState.state = STATE_SILENT;
                            m_writeRawPcm = false;
                        }
                    }
                } else { // rmsPercent >= silenceThresholdPercent
                    if (rmsCounter <= m_detectorSuccession) {
                        rmsCounter++;
                        if (rmsCounter == m_detectorSuccession && currentState.state == STATE_SILENT) {
                            currentState.state = STATE_SIGNAL;
                            m_writeRawPcm = true;
                        }
                    }
                }

                if (m_detectorStateChangedCB)
                    m_detectorStateChangedCB(currentState);
            }
        }));

        // Storage lambda in a thread
        m_storageWorker.reset(new std::thread([&] {
            std::unique_ptr<SampleFrame[]> buffer(new SampleFrame[m_streamStorageChunkSize]);

            AudioStreamManager::StorageState state = {0};
            bool lastWriteRawPcmState = false;

            auto st = new ServiceTracker("_observer-chunksink._tcp");

            while (!m_terminateRequest) {
                size_t i=0;
                while (i<m_streamStorageChunkSize && !m_terminateRequest) {
                    auto ret = m_storageBuffer->wait_dequeue_timed(buffer[i], std::chrono::milliseconds(500));            
                    if (m_terminateRequest) return;
                    if (!ret)
                        continue;

                    i++;
                }

                if (m_writeRawPcm) {

                    state.totalChunks++;

                    auto timeStampEpoche = std::chrono::system_clock::now();
                    std::stringstream ss;
                    ss << timeStampEpoche.time_since_epoch().count();
                    std::string strTimestamp = ss.str();

                    char strChunkCount[24];
                    sprintf(strChunkCount, "%016lu", state.totalChunks);

                    if (!lastWriteRawPcmState) {
                        lastWriteRawPcmState = true;
                        state.sessionID = strTimestamp;
                    }
          
                    std::string fileName = m_streamPcmOutPrefix + "_" + state.sessionID + "_" + std::string(strChunkCount) + "_" + strTimestamp + ".raw";
                    auto services = st->GetServiceMap();
                    for (const auto service : services) {
                        for (const auto se : service.second) {
                            std::string url = se.second.address + ":" + std::to_string(se.second.port);
                            std::cout << "Try url: " << url << std::endl;
                            if (sendBuffer(url, fileName, buffer.get(), m_streamStorageChunkSize))
                                break;
                        }
                    }

                    // Filesystem (rework!)
                    // ##########
                    # if 0
                    try {
                        std::string path = m_storageOutDir + "/" + fileName;
                        File fifoFd(path, O_WRONLY | O_CREAT, 0666);
                        ssize_t written = ::write(fifoFd, buffer.get(), m_streamStorageChunkSize * sizeof(SampleFrame));
                        if (written < 0) {
                            throw std::runtime_error(std::string("Error writing to file: (") + ss.str() + ") - " + strerror(errno));
                        }
                        state.totalBytes += written;

                        if (static_cast<unsigned long>(written) < m_streamStorageChunkSize) {
                            std::cerr << "Less written than expected: " << written << " instead of " << m_streamStorageChunkSize << std::endl;
                        }

                        if (m_storageChangedCB)
                            m_storageChangedCB(state);

                    } catch (std::exception &e) {
                        std::cerr << e.what() << std::endl;
                        continue;
                    }
                    #endif

                } else {
                    if (lastWriteRawPcmState)
                      lastWriteRawPcmState = false;
                }
            }
        }));
    }
}




void AudioStreamManager::stop()
{
    if (!m_terminateRequest) {

        m_terminateRequest.store(true);

        if (m_streamWorker && m_streamWorker->joinable()) {
            m_streamWorker->join();
            m_streamWorker.reset();
        }

        if (m_detectorWorker && m_detectorWorker->joinable()) {
            m_detectorWorker->join();
            m_detectorWorker.reset();
        }

        if (m_storageWorker && m_storageWorker->joinable()) {
            m_storageWorker->join();
            m_storageWorker.reset();
        }

        m_alsaAudioInput.reset();
        m_streamBuffer.reset();
        m_detectorBuffer.reset();
        m_storageBuffer.reset();
    }
}

size_t AudioStreamManager::fifoSize(int fd)
{
    int ret = fcntl(fd, F_GETPIPE_SZ);
    if (ret < 0)
        throw std::invalid_argument(std::string("Can not get fifo size. ") + m_streamFifo + ": " + strerror(errno));

    return static_cast<size_t>(ret);
}

int AudioStreamManager::fifoBytesAvailable(int fd)
{
    int nbytes = 0;
    int ret = ::ioctl(fd, FIONREAD, &nbytes);
    if (ret < 0)
        throw std::invalid_argument(std::string("Can not determin available bytes to read. ") + m_streamFifo + ": " + strerror(errno));

    return ret;
}

bool AudioStreamManager::isFifo(int fd)
{
    bool ret = true;
    struct stat st;

    if (fstat(fd, &st) < 0)
        throw std::invalid_argument(strOptStreamManagerFifo + ": " + strerror(errno));

    if (!S_ISFIFO(st.st_mode))
        ret = false;

    return ret;
}

bool AudioStreamManager::isValidPath(const std::string &path)
{
    bool ret = true;

    int fd = ::open(path.c_str(), O_RDWR);
    if (fd < 0)
        ret = false;
    else
        close(fd);

    return ret;
}
    
void AudioStreamManager::createFifo(const std::string &path)
{
    int ret = ::mkfifo(path.c_str(), 0777);
    if (ret < 0)
        throw std::invalid_argument("Can not create fifo " + path + ": " + strerror(errno));
}

size_t AudioStreamManager::setFifoSize(int fd, size_t s)
{
    int ret = fcntl(fd, F_SETPIPE_SZ, s);
    if (ret < 0)
        throw std::invalid_argument(std::string("Can not set fifo size. ") + m_streamFifo + ": " + strerror(errno));

    return fifoSize(fd);
}

bool AudioStreamManager::fifoHasReader(const std::string &path)
{
    int fd = ::open(path.c_str(), O_WRONLY | O_NONBLOCK);
    if (fd < 0)
        return false;

    close(fd);
    return true;
}
