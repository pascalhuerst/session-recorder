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

#include <iostream>
#include <fstream>
#include <chrono>
#include <thread>
#include <limits>
#include <vector>
#include <algorithm>

#include <boost/program_options.hpp>
namespace po = boost::program_options;

#include <sys/stat.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <errno.h>
#include <alsa/asoundlib.h>
#include <stdint.h>

#include "types.h"
#include "params.h"

#include "publishZeroConf/publishmDNS.h"
#include "AudioStreamManager.h"
#include "InputKey.h"
#include "Led.h"

#include "ServiceTracker.h"


void usage [[noreturn]] (const po::options_description &od)
{
    std::cout << od << std::endl;
    exit(EXIT_FAILURE);
}

int main(int argc, char **argv)
{
    // Is used in catch blocks
    po::options_description odCombined(argv[0]);

    try {
        // ########## General Options ##########
        po::options_description odGeneric("Generic");
        odGeneric.add_options()
                (strOptGenericHelp.c_str(), "Print help message")
                (strOptGenericDaemon.c_str(), "Daemonize after startup")
                (strOptGenericKillDaemon.c_str(), "Kill a running daemon")
                (strOptRecorderID.c_str(), po::value<std::string>(), "Unique ID of this recorder")
                (strOptRecorderName.c_str(), po::value<std::string>(), "Name of this recorder");


        // ########## Audio Options ##########
        po::options_description odAudio("Audio");
        odAudio.add_options()
                (strOptAudioDevice.c_str(), po::value<std::string>()->default_value("pulse"), "Alsa device to read from")
                (strOptAudioRate.c_str(), po::value<unsigned int>()->default_value(48000), "Sample rate")
                (strOptAudioChannels.c_str(), po::value<unsigned int>()->default_value(2), "Channels")
                (strOptAudioLatency.c_str(), po::value<double>()->default_value(100.0), "Input latency in ms")
                (strOptAudioFormat.c_str(), po::value<std::string>()->default_value("S16_LE"), "Sample format");

        // ########## StreamManager Options ##########
        po::options_description odStreamManager("StreamManager");
        odStreamManager.add_options()
                (strOptStreamManagerFifo.c_str(), po::value<std::string>()->default_value("/tmp/stream_pipe"), "Named pipe to write raw samples to")
                (strOptStreamManagerStreamBufferSize.c_str(), po::value<unsigned int>()->default_value(4096), "Ring buffer size in frames for pipe stream")
                (strOptStreamManagerStorageOutputDir.c_str(), po::value<std::string>(), "Directory to store raw pcm data chunks to")
                (strOptStreamManagerPcmOutChunkSize.c_str(), po::value<unsigned long>()->default_value(1024*64), "Chunk size for raw pcm files");

        // ########## Detector Options ##########
        po::options_description odDetector("Detector");
        odDetector.add_options()
                (strOptDetectorTotalTime.c_str(), po::value<double>()->default_value(5.0), "Total time a signal needs to rest before a status change occurs")
                (strOptDetectorWindowTime.c_str(), po::value<double>()->default_value(1.0), "Time window used to analize input signal")
                (strOptDetectorThreshold.c_str(), po::value<double>()->default_value(0.001), "Threshold in percent of rms signal per window to detect silence");

        // ########## LED Options ##########
        po::options_description odLed("Led");
        odLed.add_options()
                (strOptLedDetector.c_str(), po::value<std::string>(), "Use this led for detector state")
                (strOptLedIndexer.c_str(), po::value<std::string>(), "Use this led for indexer state");


        // ########## Combined ##########
        odCombined.add(odGeneric).add(odAudio).add(odStreamManager).add(odDetector).add(odLed);
        po::variables_map vmCombined;
        po::store(po::parse_command_line(argc, argv, odCombined), vmCombined);
        po::notify(vmCombined);

        if (vmCombined.count(strOptGenericHelp)) {
            usage(odCombined);
        }

        std::unique_ptr<Led> detectorLed = nullptr;
        if (vmCombined.count(strOptLedDetector)) {
            detectorLed = Led::create(vmCombined[strOptLedDetector].as<std::string>());
            detectorLed->on();
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
            detectorLed->off();
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
            detectorLed->on();
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
            detectorLed->off();
        }

        std::unique_ptr<Led> indexerLed = nullptr;
        if (vmCombined.count(strOptLedIndexer)) {
            indexerLed = Led::create(vmCombined[strOptLedIndexer].as<std::string>());
            indexerLed->on();
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
            indexerLed->off();
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
            indexerLed->on();
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
            indexerLed->off();
        }


        AudioStreamManager streamManager(vmCombined,
        
            [&](AudioStreamManager::DetectorState s) {
                std::cout << "detector:   " << s.rmsPercent << "  " 
                          << (s.state == AudioStreamManager::STATE_SIGNAL ? "Signal" : "Silence") << std::endl;
                if (detectorLed) {
                    if (s.state == AudioStreamManager::STATE_SIGNAL) {
                        detectorLed->on();
                    } else {
                        detectorLed->off();
                        if (indexerLed) {
                            indexerLed->off();
                        }
                    }
                }
            },

            [&](AudioStreamManager::StorageState s){

                static bool toggle = false;
                toggle = !toggle;

                std::cout << "storage: totalBytes=" << s.totalBytes << std::endl
                          << "            totalChunks=" << s.totalChunks << std::endl
                          << "            sessionCount=" << s.sessionID << std::endl;

                if (indexerLed) {
                    if (toggle) {
                        indexerLed->on();
                    } else {
                        indexerLed->off();
                    }
                }
            }
        );

        streamManager.start();
        while(getchar() != 'q') {
            std::this_thread::sleep_for(std::chrono::milliseconds(1000));
        }
        streamManager.stop();

    } catch (std::invalid_argument &e) {

        std::cerr << odCombined << "\n";
        std::cerr << e.what() << std::endl;
        return -EINVAL;

    } catch (std::runtime_error &e) {

        std::cerr << e.what() << std::endl;
        return -EIO;


    } catch (std::exception &e) {
        std::cerr << e.what() << std::endl;
        return -EINVAL;
    }

    return EXIT_SUCCESS;
}
