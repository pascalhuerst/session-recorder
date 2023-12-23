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

#include "Led.h"
#include <exception>
#include <iterator>
#include <algorithm>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>

const std::string Led::s_basePath = "/sys/class/leds";

Led::Led(const std::string& key) :
    m_key(key),
    m_fd(open(Led::s_basePath + "/" + m_key + "/brightness")),
    m_maxBrightness(maxBrightness(Led::s_basePath + "/" + m_key + "/max_brightness"))
{
}

Led::~Led()
{
    close(m_fd);
}

//static
std::unique_ptr<Led> Led::create(const std::string &key)
{
    struct EnableMaker : public Led { EnableMaker(const std::string& key) : Led(key) { } };
    return std::make_unique<EnableMaker>(key);
}

std::string Led::info()
{
    std::string ret = m_key + std::string(": max_brightness: ") + std::to_string(m_maxBrightness) + std::string(" | fd: ") + std::to_string(m_fd);

    return ret;
}

int Led::maxBrightness()
{
    return m_maxBrightness;
}

void Led::on()
{
    set(m_maxBrightness);
}

void Led::off()
{
    set(0);
}

void Led::set(int value)
{
    if (value < 0) value = 0;
    if (value > m_maxBrightness) value = m_maxBrightness;
    std::string strVal = std::to_string(value);
    ssize_t ret = ::write(m_fd, strVal.c_str(), strVal.length());
    if (ret < 0)
        throw std::runtime_error(std::string("Can not set brightness: ") + m_key + " - " + strerror(errno));

    return;

}

int Led::open(const std::string& path)
{
    int fd = ::open(path.c_str(), O_WRONLY);
    if (fd < 0)
        throw std::runtime_error(std::string("Can not open led: ") + path + " - " + strerror(errno));

    return fd;
}

int Led::maxBrightness(const std::string &path)
{
    int fd = ::open(path.c_str(), O_RDONLY);
    if (fd < 0)
        throw std::runtime_error(std::string("Can not open max_brightness: ") + path + " - " + strerror(errno));

    char buf[16];
    ssize_t ret = read(fd, buf, 16);
    int cpErrno = errno;
    close(fd);

    if (ret < 0)
        throw std::runtime_error(std::string("Can not read max_brightness: ") + path + " - " + strerror(cpErrno));

    return atoi(buf);
}
