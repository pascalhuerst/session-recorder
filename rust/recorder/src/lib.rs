pub mod audio {
    pub mod alsa;
    pub mod callback_thread;
    pub mod channels;
    pub mod generic_buffer;
    pub mod utils;
}

pub mod io {
    pub mod input_key;
    pub mod led;
}

pub mod grpc {
    pub mod chunk_sink_client;
}

pub mod discovery;

pub use audio::*;
pub use grpc::*;
pub use io::*;
