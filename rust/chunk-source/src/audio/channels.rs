use ringbuf::{HeapRb, traits::*};

pub type Sample = i16;

type AudioRingBuffer = HeapRb<Sample>;
pub type AudioRingBufferConsumer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<AudioRingBuffer>, false, true>;
pub type AudioRingBufferProducer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<AudioRingBuffer>, true, false>;

pub struct AudioChannels {
    producer: Option<AudioRingBufferProducer>,
    pub consumer: AudioRingBufferConsumer,
}

impl AudioChannels {
    pub fn new(buffer_size: usize) -> Self {
        let audio_rb = HeapRb::<Sample>::new(buffer_size);
        let (audio_producer, audio_consumer) = audio_rb.split();

        Self {
            producer: Some(audio_producer),
            consumer: audio_consumer,
        }
    }

    pub fn producer_mut(&mut self) -> Option<&mut AudioRingBufferProducer> {
        self.producer.as_mut()
    }

    pub fn take_producer(&mut self) -> Option<AudioRingBufferProducer> {
        self.producer.take()
    }
}

#[derive(Debug, Clone, Copy)]
pub enum Parameters {
    Cut(),
    Shutdown(),
}

pub type ParameterRingBuffer = HeapRb<Parameters>;
pub type ParameterRingBufferConsumer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<ParameterRingBuffer>, false, true>;
pub type ParameterRingBufferProducer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<ParameterRingBuffer>, true, false>;

pub struct ParameterChannels {
    pub consumer: ParameterRingBufferConsumer,
    pub producer: ParameterRingBufferProducer,
}

impl ParameterChannels {
    pub fn new(buffer_size: usize) -> Self {
        let input_rb = HeapRb::<Parameters>::new(buffer_size);
        let (parameter_producer, parameter_consumer) = input_rb.split();

        Self {
            consumer: parameter_consumer,
            producer: parameter_producer,
        }
    }
}
