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
#include <list>
#include <atomic>
#include <thread>
#include <chrono>

#include "types.h"

class Led {

public:
    ~Led();

    static std::list<std::string> available();
    static std::unique_ptr<Led> create(const std::string &key);

    Led(const Led&) = delete; // non construction-copyable
    Led& operator=(const Led&) = delete; // non copyable

    std::string info();
    int maxBrightness();

    void on();
    void off();
    void set(int value);

private:
    const static std::string s_basePath;

    Led(const std::string &key);

    int open(const std::string& path);
    int maxBrightness(const std::string& path);

    std::string m_key;
    int m_fd;
    int m_maxBrightness;
};
