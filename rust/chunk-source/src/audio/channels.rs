use ringbuf::{traits::*, HeapRb};

type AudioRingBuffer = HeapRb<f32>;
pub type AudioRingBufferConsumer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<AudioRingBuffer>, false, true>;
pub type AudioRingBufferProducer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<AudioRingBuffer>, true, false>;

pub struct AudioChannels {
    pub input_producer: AudioRingBufferProducer,
    pub output_consumer: AudioRingBufferConsumer,
}

pub struct AudioChannelPair {
    pub callback_channels: AudioChannels,
    pub main_channels: AudioChannels,
}

impl AudioChannelPair {
    pub fn new(buffer_size: usize) -> Self {
        let input_rb = HeapRb::<f32>::new(buffer_size);
        let (input_producer, input_consumer) = input_rb.split();

        let output_rb = HeapRb::<f32>::new(buffer_size);
        let (output_producer, output_consumer) = output_rb.split();

        Self {
            callback_channels: AudioChannels {
                input_producer,
                output_consumer,
            },
            main_channels: AudioChannels {
                input_producer: output_producer,
                output_consumer: input_consumer,
            },
        }
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
    pub input_consumer: ParameterRingBufferConsumer,
    pub output_producer: ParameterRingBufferProducer,
}

impl ParameterChannels {
    pub fn new(buffer_size: usize) -> Self {
        let rb = HeapRb::<Parameters>::new(buffer_size);
        let (output_producer, input_consumer) = rb.split();

        Self {
            input_consumer,
            output_producer,
        }
    }
}
