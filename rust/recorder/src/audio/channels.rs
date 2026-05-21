use ringbuf::{HeapRb, traits::*};

type AudioRingBuffer = HeapRb<f32>;
pub type AudioRingBufferConsumer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<AudioRingBuffer>, false, true>;
pub type AudioRingBufferProducer =
    ringbuf::wrap::caching::Caching<std::sync::Arc<AudioRingBuffer>, true, false>;

pub struct CaptureProducer {
    pub producer: AudioRingBufferProducer,
}

pub struct CaptureConsumer {
    pub consumer: AudioRingBufferConsumer,
}

pub fn new_capture_ring(buffer_size: usize) -> (CaptureProducer, CaptureConsumer) {
    let rb = HeapRb::<f32>::new(buffer_size);
    let (producer, consumer) = rb.split();
    (CaptureProducer { producer }, CaptureConsumer { consumer })
}
