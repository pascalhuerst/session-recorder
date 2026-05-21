use alsa::pcm::{Access, Format, HwParams};
use alsa::{Direction, PCM};
use anyhow::Result;

#[derive(Debug, Default, Clone)]
pub struct AudioSettings {
    pub input_device: String,
    pub num_channels: u32,
    pub period_size: u32,
    pub buffer_size: u32,
    pub sample_rate: u32,
}

pub fn configure_input_device(audio_settings: &AudioSettings) -> Result<PCM> {
    let capture = PCM::new(&audio_settings.input_device, Direction::Capture, false)
        .expect("Failed to create input PCM device");
    configure_pcm(
        &capture,
        audio_settings.num_channels,
        audio_settings.sample_rate,
        audio_settings.buffer_size,
        audio_settings.period_size,
    )?;
    Ok(capture)
}

pub fn configure_pcm(
    pcm: &PCM,
    num_channels: u32,
    sample_rate: u32,
    buffer_size: u32,
    period_size: u32,
) -> Result<()> {
    let hwp = HwParams::any(pcm)?;
    hwp.set_channels(num_channels)?;
    hwp.set_rate(sample_rate, alsa::ValueOr::Nearest)?;
    hwp.set_format(Format::S16LE)?;
    hwp.set_access(Access::RWInterleaved)?;

    // alsa::pcm::Frames is the C `snd_pcm_sframes_t` (a `long`): i64 on 64-bit
    // targets, i32 on 32-bit ones. Cast to Frames, not a fixed-width int, so
    // this compiles on both.
    hwp.set_buffer_size_near(buffer_size as alsa::pcm::Frames)?;
    hwp.set_period_size_near(period_size as alsa::pcm::Frames, alsa::ValueOr::Nearest)?;

    pcm.hw_params(&hwp)?;

    Ok(())
}

pub fn get_buffer_size(pcm: &PCM) -> Result<usize> {
    match pcm.hw_params_current() {
        Ok(hwparams) => {
            if let Ok(reported_buffer_size) = hwparams.get_buffer_size() {
                Ok(reported_buffer_size as usize)
            } else {
                Err(anyhow::anyhow!("Failed to get buffer size"))
            }
        }
        _ => Err(anyhow::anyhow!("Failed to get buffer size")),
    }
}

pub fn get_period_size(pcm: &PCM) -> Result<usize> {
    match pcm.hw_params_current() {
        Ok(hwparams) => {
            if let Ok(reported_period_size) = hwparams.get_period_size() {
                Ok(reported_period_size as usize)
            } else {
                Err(anyhow::anyhow!("Failed to get period size"))
            }
        }
        _ => Err(anyhow::anyhow!("Failed to get period size")),
    }
}
