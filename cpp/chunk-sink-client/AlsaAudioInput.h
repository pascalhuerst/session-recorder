/*
 * Copyright (C) 2018 Pascal Huerst <pascal.huerst@gmail.com>
 * Inspired by work of Henry Hoegelow
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

#include <alsa/asoundlib.h>
#include <functional>
#include <thread>
#include <atomic>

#include <boost/program_options.hpp>
namespace po = boost::program_options;

#include "types.h"

class AlsaAudioInput {

public:
    using Callback = std::function<void(SampleFrame *frames, size_t numFrames)>;

    AlsaAudioInput(const po::variables_map &vmAudio, Callback cb);
    ~AlsaAudioInput();

    std::chrono::nanoseconds latency() const;
    snd_pcm_uframes_t framesProcessed() const;

    void prioritizeThread();

private:
    void open(const std::string &deviceName, unsigned int rate, unsigned int channels, snd_pcm_format_t format,  double latency = 20.0 /* milliseconds */);
    void start();
    void close();
    void workerThread();
    void handleReadError(snd_pcm_sframes_t result);
    void setThreadAffinity();

    Callback m_cb;
    std::atomic<bool> m_isRunning;

    snd_pcm_t *m_handle = nullptr;
    std::thread m_workerThread;
    unsigned int m_latency = 0;
    snd_pcm_uframes_t m_framesPerPeriod = 0;
    snd_pcm_uframes_t m_ringBufferFrames = 0;
    std::atomic<snd_pcm_uframes_t> m_framesProcessed;
};
