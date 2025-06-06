// Date: 2021-09-26
// Name: RecorderStatus

use grpc_test::common::*;
use protobuf::EnumOrUnknown;

fn main() {
    let mut status = RecorderStatus::new();
    status.clipping = true;
    status.recorderID = "42".to_string();
    status.signalStatus = EnumOrUnknown::try_from(SignalStatus::NO_SIGNAL).unwrap();

    println!("Hello, world! {:?}", status);
}
