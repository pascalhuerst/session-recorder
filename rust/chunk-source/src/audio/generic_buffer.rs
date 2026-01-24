use num_traits::{NumCast, Zero};

use crate::channels::Sample;

#[derive(Clone, Debug)]
pub struct GenericBuffer<T>
where
    T: Copy + Zero + NumCast,
{
    samples: Box<[T]>,
    num_channels: usize,
    buffer_size: usize,
}

impl<T> GenericBuffer<T>
where
    T: Copy + Zero + NumCast,
{
    pub fn new(num_channels: usize, buffer_size: usize) -> Self {
        let num_samples = num_channels * buffer_size;
        if num_samples == 0 {
            panic!("Cannot create an AudioBuffer with zero samples");
        }
        let samples = vec![T::zero(); num_samples].into_boxed_slice();
        Self {
            samples,
            num_channels,
            buffer_size,
        }
    }

    pub fn num_channels(&self) -> usize {
        self.num_channels
    }

    pub fn buffer_size(&self) -> usize {
        self.buffer_size
    }

    #[inline(always)]
    pub fn clear(&mut self) {
        self.samples.fill(T::zero());
    }

    // assumes non-interleaved layout of samples in buffer
    #[inline(always)]
    pub fn get(&self, channel: usize) -> &[T] {
        let start = channel * self.buffer_size;
        let end = start + self.buffer_size;
        &self.samples[start..end]
    }

    // assumes non-interleaved layout of samples in buffer
    #[inline(always)]
    pub fn get_mut(&mut self, channel: usize) -> &mut [T] {
        let start = channel * self.buffer_size;
        let end = start + self.buffer_size;
        &mut self.samples[start..end]
    }

    #[inline(always)]
    pub fn get_all(&self) -> &[T] {
        &self.samples[..self.num_channels * self.buffer_size]
    }

    #[inline(always)]
    pub fn get_all_mut(&mut self) -> &mut [T] {
        &mut self.samples[..self.num_channels * self.buffer_size]
    }
}

#[inline(always)]
pub fn copy_from_slice(src: &[f32], dst: &mut [f32]) {
    dst.copy_from_slice(src);
}

#[inline(always)]
pub fn get_abs_peak(buffer: &[f32]) -> f32 {
    buffer.iter().map(|&x| x.abs()).fold(0., f32::max)
}

#[inline(always)]
pub fn zero_slice(buffer: &mut [f32]) {
    for sample in buffer.iter_mut() {
        *sample = 0.;
    }
}

//pub type IntBuffer = GenericBuffer<Sample>;
pub type AudioBuffer = GenericBuffer<Sample>;
