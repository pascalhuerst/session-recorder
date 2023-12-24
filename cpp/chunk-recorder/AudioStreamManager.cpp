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
#include "httplib.h"

#include <grpc/grpc.h>

#include <grpcpp/create_channel.h>
#include "chunksink.pb.h"
#include "chunksink.grpc.pb.h"
#include "common.pb.h"

#include <sys/ioctl.h>
#include <signal.h>
#include <stdint.h>
#include <sys/time.h>
#include <poll.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/stat.h>

#include <boost/uuid/uuid.hpp>
#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>


AudioStreamManager::AudioStreamManager(const po::variables_map &vmCombined,
                                       CallbackDetector onDetectorStateChangedCB,
                                       CallbackStorage onStorageChangedCB) :
    m_recorderID(""),
    m_recorderName(""),
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
    m_streamStorageChunkSize(0),
    m_detectorStateChangedCB(onDetectorStateChangedCB),
    m_storageChangedCB(onStorageChangedCB),
    m_serviceTracker(nullptr)
{
}

AudioStreamManager::~AudioStreamManager()
{
}

bool sendChunks(const std::string &url, chunksink::Chunks &chunks) {
    auto channel = grpc::CreateChannel(url, grpc::InsecureChannelCredentials());

    auto stub_ = chunksink::ChunkSink::NewStub(channel);

    grpc::ClientContext context;
    common::Respone response;
    
    grpc::Status status = stub_->SetChunks(&context, chunks, &response);

    if (!status.ok()) {
        return false;
    }

    if (response.success() != true) {
        std::cerr << "Error sending chunks to: " << url << " -- " <<  response.errormessage() << std::endl;
        
        return false;
    }

    return true;
}

bool sendDetectorStatus(const std::string &url, common::RecorderStatus &recorderStatus) {
    auto channel = grpc::CreateChannel(url, grpc::InsecureChannelCredentials());

    auto stub_ = chunksink::ChunkSink::NewStub(channel);

    grpc::ClientContext context;
    common::Respone response;
    
    grpc::Status status = stub_->SetRecorderStatus(&context, recorderStatus, &response);

    if (!status.ok()) {
        return false;
    }

    if (response.success() != true) {
        std::cerr << "Error sending detector status to: " << url << " -- " <<  response.errormessage() << std::endl;
        
        return false;
    }

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

        if (m_vmCombined.count(strOptRecorderID)) {
            m_recorderID = m_vmCombined[strOptRecorderID].as<std::string>();
        } else {
            throw std::invalid_argument(strOptRecorderID + " must be set!");
        }

        if (m_vmCombined.count(strOptRecorderName)) {
            m_recorderName = m_vmCombined[strOptRecorderName].as<std::string>();
        } else {
            throw std::invalid_argument(strOptRecorderName + " must be set!");
        }

        if (m_vmCombined.count(strOptStreamManagerStorageOutputDir)) {
            m_storageOutDir = m_vmCombined[strOptStreamManagerStorageOutputDir].as<std::string>();
        } else {
            throw std::invalid_argument(strOptStreamManagerStorageOutputDir + " must be set!");
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

        //TODO; This crashed when destroyed
        m_serviceTracker.reset(new ServiceTracker("_session-recorder-chunksink._tcp"));


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

                bool clipping = false;

                double chunkSum = 0.0;
                for (unsigned int i=0; i<m_detectorBufferSize; ++i) {

                    double monoSum = (buffer[i].left + buffer[i].right) / 2.0;
                    chunkSum += (monoSum * monoSum);

                    if (buffer[i].left == std::numeric_limits<Sample>::max() || 
                        buffer[i].right == std::numeric_limits<Sample>::max() ||
                        buffer[i].left == std::numeric_limits<Sample>::min() ||
                        buffer[i].right == std::numeric_limits<Sample>::min()) {
                        clipping = true;
                    }
                }

                std::cout << "clipping: " << clipping << std::endl;

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

                common::RecorderStatus recorderStatus;
                recorderStatus.set_recorderid(m_recorderID);
                recorderStatus.set_recordername(m_recorderName);
                recorderStatus.set_signalstatus(currentState.state == STATE_SIGNAL ? common::SignalStatus::SIGNAL : common::SignalStatus::NO_SIGNAL);
                recorderStatus.set_rmspercent(currentState.rmsPercent);
                recorderStatus.set_clipping(clipping);

                auto services = m_serviceTracker->GetServiceMap();
                for (const auto &service : services) {
                    for (const auto &se : service.second) {
                        std::string url = se.second.address + ":" + std::to_string(se.second.port);
                        std::cout << "Send detector using: " << url; 
                        if (sendDetectorStatus(url, recorderStatus)) {
                            std::cout << "  OK" << std::endl;

                            break;
                        }

                        std::cout << "  Failed" << std::endl;
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

                    if (!lastWriteRawPcmState) {
                        lastWriteRawPcmState = true;

                        boost::uuids::uuid uuid = boost::uuids::random_generator()();
                        state.sessionID = boost::uuids::to_string(uuid);
                        state.startTime = google::protobuf::util::TimeUtil::GetCurrentTime();
                        state.totalChunks = 0;
                    }
          
                    chunksink::Chunks chunks;
                    chunks.set_recorderid(m_recorderID);
                    chunks.set_sessionid(state.sessionID);
                    chunks.set_chunkcount(state.totalChunks);
                    chunks.set_allocated_timecreated(new google::protobuf::Timestamp(state.startTime));
                    
                    auto rawData = buffer.get();
                    for (size_t i=0; i<m_streamStorageChunkSize; i++) {
                        chunks.add_data(static_cast<uint32_t>(rawData[i].left));
                        chunks.add_data(static_cast<uint32_t>(rawData[i].right));
                    }

                    auto services = m_serviceTracker->GetServiceMap();
                    for (const auto &service : services) {
                        for (const auto &se : service.second) {
                            std::string url = se.second.address + ":" + std::to_string(se.second.port);
                            std::cout << "Send chunks using: " << url;
                              
                            if (sendChunks(url, chunks)) {
                                 std::cout << "  OK" << std::endl;

                                break;
                            }

                            std::cout << "  Failed" << std::endl;
                        }
                    }
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
