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

#include "InputKey.h"
#include <exception>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <linux/input.h>
#include <poll.h>
#include <unistd.h>

InputKey::InputKey(const po::variables_map &vmGpioKey, int inputNr) :
    m_pressValue(1),
    m_releaseValue(0),
    m_mutex(),
    m_eventMap(),
    m_isRunning(false),
    m_terminateRequest(false)
{
    (void)vmGpioKey;
    start(inputNr);
}

InputKey::~InputKey()
{
    stop();
}

void InputKey::start(int inputNr)
{
    if (!m_isRunning) {
        m_isRunning = true;
        m_workerThread = std::thread([=]() { workerThread(inputNr); });
    }
}

void InputKey::stop()
{
    if (m_isRunning) {
        m_isRunning = false;

        m_terminateRequest = true;

        if (m_workerThread.joinable())
            m_workerThread.join();
    }
}

int InputKey::openInputDev(int nr)
{
    int  fd;
    char tmp[128];

    snprintf(tmp, sizeof(tmp), "/dev/input/event%d", nr);
    fd = open(tmp, O_RDONLY);
    if (fd < 0)
        throw std::runtime_error("Can not open " + std::string(tmp) + ": " + strerror(errno));

    return fd;
}

int InputKey::diffInMs(struct timeval* start, struct timeval* stop)
{
    struct timeval diff;

    diff.tv_sec  = stop->tv_sec - start->tv_sec;
    diff.tv_usec = stop->tv_usec - start->tv_usec;

    return static_cast<int>(diff.tv_sec * 1000) + static_cast<int>(diff.tv_usec / 1000);
}

void InputKey::workerThread(int inputNr)
{
    const int timeout = 500;

    try {
        int fd = openInputDev(inputNr);

        while (!m_terminateRequest) {

            struct input_event ev;
            struct pollfd pfd;

            pfd.fd = fd;
            pfd.events = POLLIN;

            ssize_t ret = poll(&pfd, 1, timeout);

            if (ret == 0) { //poll timeout
                continue;

            } else if (ret > 0) { //poll ready to read

                ret = read(fd, &ev, sizeof(ev));

                if (ret < 0)
                    throw std::runtime_error(std::string("Error while reading input device: ") + strerror(errno));

                if (ev.type != EV_KEY)
                    continue;

                { // Block for RAII (unique_lock)
                    std::unique_lock<std::mutex> mlock(m_mutex);
                    int code = static_cast<int>(ev.code);
                    if (m_eventMap.count(code)) {
                        auto &event = m_eventMap[code];
                        if (ev.value == m_pressValue) { // press
                            struct timeval now = {0, 0};
                            gettimeofday(&now, nullptr);
                            event.timestampPress = now;
                            event.pc();
                        } else if (ev.value == m_releaseValue) { // release
                            struct timeval now = {0, 0};
                            gettimeofday(&now, nullptr);
                            int millis = diffInMs(&event.timestampPress, &now);
                            event.rc(std::chrono::milliseconds(millis));
                            event.timestampPress = {0, 0};
                        } else { // If key is kept pressed, value becomes 2 -> not interesting here
                            continue;
                        }
                    }
                }

            } else {
                throw std::runtime_error(std::string("poll() returned error: ") + strerror(errno));
            }
        }

        close(fd);

    } catch (std::exception &e) {

        std::cerr << e.what() << std::endl;
        return;
    }



}

void InputKey::registerKey(int key, PressCallback pc, ReleaseCallback rc)
{
    Event e;
    e.pc = pc;
    e.rc = rc;
    e.timestampPress = {0, 0};

    std::unique_lock<std::mutex> mlock(m_mutex);
    m_eventMap.emplace(std::make_pair(key, e));
}
