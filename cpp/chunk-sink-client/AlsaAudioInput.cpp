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

#include "AlsaAudioInput.h"
#include "params.h"

#include <utility>
#include <chrono>
#include <exception>

#include <boost/program_options.hpp>
namespace po = boost::program_options;

AlsaAudioInput::AlsaAudioInput(const po::variables_map &vmAudio, Callback cb) :
    m_cb(cb),
    m_isRunning(false)
{
    std::string device = "";
    if (vmAudio.count(strOptAudioDevice)) {
        device = vmAudio[strOptAudioDevice].as<std::string>();
    } else {
        throw std::invalid_argument(strOptAudioDevice + "must be set!");
    }

    unsigned int channels = 0;
    if (vmAudio.count(strOptAudioChannels)) {
        channels = vmAudio[strOptAudioChannels].as<unsigned int>();
    } else {
        throw std::invalid_argument(strOptAudioChannels + "must be set!");
    }

    snd_pcm_format_t format;
    if (vmAudio.count(strOptAudioFormat)) {
        auto strFormat = vmAudio[strOptAudioFormat].as<std::string>();
        if (strFormat != "S16_LE")
            throw std::invalid_argument(strOptAudioFormat + ": Only supported format right now is S16_LE.");

        format = SND_PCM_FORMAT_S16;
    } else {
        throw std::invalid_argument(strOptAudioFormat + "must be set!");
    }

    unsigned int rate = 0;
    if (vmAudio.count(strOptAudioRate)) {
        rate = vmAudio[strOptAudioRate].as<unsigned int>();
    } else {
        throw std::invalid_argument(strOptAudioRate + "must be set!");
    }

    double latency = 0.0;
    if (vmAudio.count(strOptAudioLatency)) {
        latency = vmAudio[strOptAudioLatency].as<double>();
    } else {
        throw std::invalid_argument(strOptAudioLatency + "must be set!");
    }

    open(device, rate, channels, format, latency);
    start();
}

AlsaAudioInput::~AlsaAudioInput()
{
    close();
}

std::chrono::nanoseconds AlsaAudioInput::latency() const
{
    return std::chrono::microseconds(static_cast<uint64_t>(m_latency));
}

snd_pcm_uframes_t AlsaAudioInput::framesProcessed() const
{
    return m_framesProcessed;
}

void AlsaAudioInput::open(const std::string &deviceName, unsigned int rate, unsigned int channels, snd_pcm_format_t format, double latency)
{
    snd_pcm_open(&m_handle, deviceName.c_str(), SND_PCM_STREAM_CAPTURE, SND_PCM_ASYNC);

    snd_pcm_hw_params_t* hwparams = nullptr;
    snd_pcm_sw_params_t* swparams = nullptr;

    snd_pcm_hw_params_alloca(&hwparams);
    snd_pcm_sw_params_alloca(&swparams);

    unsigned int sampleRate = rate;
    unsigned int periods = 4;

    auto timePerPeriod = 1.0 * latency / (periods + 1);

    m_framesPerPeriod = static_cast<unsigned int>(timePerPeriod) * sampleRate / 1000;
    m_ringBufferFrames = periods * m_framesPerPeriod;

    snd_pcm_hw_params_any(m_handle, hwparams);
    snd_pcm_hw_params_set_access(m_handle, hwparams, SND_PCM_ACCESS_RW_INTERLEAVED);
    snd_pcm_hw_params_set_format(m_handle, hwparams, format);
    snd_pcm_hw_params_set_channels(m_handle, hwparams, channels);
    snd_pcm_hw_params_set_rate_near(m_handle, hwparams, &sampleRate, nullptr);
    snd_pcm_hw_params_set_periods(m_handle, hwparams, periods, 0);
    snd_pcm_hw_params_set_period_size_near(m_handle, hwparams, &m_framesPerPeriod, nullptr);
    snd_pcm_hw_params_set_buffer_size_near(m_handle, hwparams, &m_ringBufferFrames);
    snd_pcm_hw_params(m_handle, hwparams);
    snd_pcm_sw_params_current(m_handle, swparams);
    snd_pcm_sw_params(m_handle, swparams);

    snd_pcm_hw_params_get_period_time(hwparams, &m_latency, nullptr);
    snd_pcm_hw_params_get_periods(hwparams, &periods, nullptr);

    m_latency *= periods + 1;
}

void AlsaAudioInput::start()
{
    if (!m_isRunning) {
        m_isRunning = true;
        m_workerThread = std::thread([this]() { workerThread(); });
    }
}

void AlsaAudioInput::close()
{
    if (m_isRunning) {

        m_isRunning = false;

        if(m_workerThread.joinable())
            m_workerThread.join();

        if(auto h = std::exchange(m_handle, nullptr))
            snd_pcm_close(h);
    }
}

void AlsaAudioInput::workerThread()
{
    std::cout << "Starting Audio Worker Thread." << std::endl;

    prioritizeThread();
    setThreadAffinity();

    snd_pcm_prepare(m_handle);

    SampleFrame samples[m_framesPerPeriod];
    std::fill(samples, samples + m_framesPerPeriod, SampleFrame{});

    snd_pcm_start(m_handle);

    while(m_isRunning)
    {
        auto ret = snd_pcm_readi(m_handle, samples, m_framesPerPeriod);
        if(static_cast<snd_pcm_uframes_t>(ret) != m_framesPerPeriod)
            handleReadError(ret);
        else
            m_framesProcessed += m_framesPerPeriod;

        m_cb(samples, m_framesPerPeriod);
    }
}

void AlsaAudioInput::prioritizeThread()
{
    struct sched_param param;
    param.sched_priority = 50;

    if(pthread_setschedparam(pthread_self(), SCHED_FIFO, &param))
        std::cerr << "Could not set thread priority - consider 'sudo setcap 'cap_sys_nice=eip' <application>'" << std::endl;
}

void AlsaAudioInput::setThreadAffinity()
{
    int coreID = 0;

    cpu_set_t set;
    CPU_ZERO(&set);
    CPU_SET(coreID, &set);
    if(sched_setaffinity(0, sizeof(cpu_set_t), &set) < 0)
        std::cerr << "Could not set thread affinity" << std::endl;
}

void AlsaAudioInput::handleReadError(snd_pcm_sframes_t result)
{
    if (result < 0) {

        if (auto recoverResult = snd_pcm_recover(m_handle, static_cast<int>(result), 1)) {
            std::cerr << "Could not recover:" << recoverResult << std::endl;
        } else {
            std::cerr << "recovered from x-run" << std::endl;
            snd_pcm_start(m_handle);
        }
    }
}
