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

#pragma once

#include <string>

// ########## General Options ##########
const std::string strOptGenericHelp = "help";
const std::string strOptRecorderID = "recorder-id";
const std::string strOptRecorderName = "recorder-name";

// ########## Audio Options ##########
const std::string strOptAudioDevice = "device";
const std::string strOptAudioChannels = "channels";
const std::string strOptAudioFormat = "format";
const std::string strOptAudioRate = "rate";
const std::string strOptAudioLatency = "latency";


// ########## Detector Options ##########
const std::string strOptDetectorTotalTime = "rest-time";
const std::string strOptDetectorWindowTime = "window-time";
const std::string strOptDetectorThreshold = "detector-threshold";

// ########## Led Options ##########
const std::string strOptLedDetector = "led-detector";
const std::string strOptLedIndexer = "led-indexer";
