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

#include <map>
#include <atomic>
#include <functional>
#include <thread>
#include <mutex>
#include <chrono>

#include <boost/program_options.hpp>
namespace po = boost::program_options;

#include "types.h"

class InputKey {

public:
    using PressCallback = std::function<void(void)>;
    using ReleaseCallback = std::function<void(std::chrono::milliseconds preeTime)>;

    struct Event {
        PressCallback pc;
        ReleaseCallback rc;
        struct timeval timestampPress;
    };


    InputKey(const po::variables_map &vmGpioKey, int inputNr);
    ~InputKey();

    void registerKey(int keyCode, PressCallback pc, ReleaseCallback rc);

private:

    void start(const int inputNr);
    void stop();
    void workerThread(int inputNr);
    int openInputDev(int nr);
    int diffInMs(struct timeval* start, struct timeval* stop);

    int32_t m_pressValue;
    int32_t m_releaseValue;

    std::mutex m_mutex;

    std::map<int, Event> m_eventMap;
    std::atomic<bool> m_isRunning;
    std::atomic<bool> m_terminateRequest;
    std::thread m_workerThread;

};
