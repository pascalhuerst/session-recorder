use num_traits::{Float, FromPrimitive, NumCast};
const BITS_32_SCALE: f32 = 2147483647.0;
const INV_BITS_32_SCALE: f32 = 1. / BITS_32_SCALE;
const MAX_FLOAT: f32 = 1. - INV_BITS_32_SCALE;
const MIN_FLOAT: f32 = -1.;

#[inline(always)]
pub fn interleave<T: Copy>(input: &[T], output: &mut [T], num_channels: usize) {
    assert_eq!(input.len(), output.len());
    assert_eq!(input.len() % num_channels, 0);
    let buffer_size = input.len() / num_channels;
    for frame in 0..buffer_size {
        for ch in 0..num_channels {
            output[frame * num_channels + ch] = input[ch * buffer_size + frame];
        }
    }
}

#[inline(always)]
pub fn interleave_and_convert_to_i32<F: Float>(
    input: &[F],
    output: &mut [i32],
    num_channels: usize,
) {
    assert_eq!(input.len(), output.len());
    assert_eq!(input.len() % num_channels, 0);
    let buffer_size = input.len() / num_channels;
    for frame in 0..buffer_size {
        for ch in 0..num_channels {
            output[frame * num_channels + ch] =
                float_to_i32_sample(input[ch * buffer_size + frame]);
        }
    }
}

#[inline(always)]
pub fn deinterleave<T: Copy>(input: &[T], output: &mut [T], num_channels: usize) {
    assert_eq!(input.len(), output.len());
    assert_eq!(input.len() % num_channels, 0);
    let buffer_size = input.len() / num_channels;
    for i in (0..input.len()).step_by(num_channels) {
        for ch in 0..num_channels {
            output[ch * buffer_size + (i / num_channels)] = input[i + ch];
        }
    }
}

#[inline(always)]
pub fn deinterleave_and_convert_to_float<F: Float>(
    input: &[i16],
    output: &mut [F],
    num_channels: usize,
) {
    assert_eq!(input.len(), output.len());
    assert_eq!(input.len() % num_channels, 0);
    let buffer_size = input.len() / num_channels;
    for i in (0..input.len()).step_by(num_channels) {
        for ch in 0..num_channels {
            output[ch * buffer_size + (i / num_channels)] = int16_to_float_sample(input[i + ch]);
        }
    }
}

#[inline(always)]
pub fn int16_to_float_sample<T: Float>(input: i16) -> T {
    let inv_scale = T::from(1.0_f32 / 32768.0).unwrap();
    T::from(input).unwrap() * inv_scale
}

#[inline(always)]
pub fn int32_to_float_sample<T: Float>(input: i32) -> T {
    let output = T::from(input).unwrap() * T::from(INV_BITS_32_SCALE).unwrap();
    output
}

#[inline(always)]
pub fn float_to_i32_sample<T: Float>(input: T) -> i32 {
    let clamped: f32 = NumCast::from(
        input
            .min(NumCast::from(MAX_FLOAT).unwrap())
            .max(NumCast::from(MIN_FLOAT).unwrap()),
    )
    .unwrap();

    let output = (clamped * BITS_32_SCALE) as i32;
    output
}

#[inline(always)]
pub fn int32_to_float<T: Float>(input: &[i32], output: &mut [T]) {
    for (input_sample, output_sample) in input.iter().zip(output.iter_mut()) {
        *output_sample = int32_to_float_sample(*input_sample);
    }
}

#[inline(always)]
pub fn float_to_int32<T: Float>(input: &[T], output: &mut [i32]) {
    for (input_sample, output_sample) in input.iter().zip(output.iter_mut()) {
        *output_sample = float_to_i32_sample(*input_sample);
    }
}

// Convert decibels to linear amplitude
pub fn db_to_linear<FloatType>(db: FloatType) -> FloatType
where
    FloatType: Float + FromPrimitive,
{
    let ten = FloatType::from_f64(10.0).unwrap();
    let factor = FloatType::from_f64(0.05).unwrap();
    ten.powf(db * factor)
}

// Convert decibels to linear amplitude with root scaling
pub fn db_to_root_linear<FloatType>(gain_db: FloatType) -> FloatType
where
    FloatType: Float + FromPrimitive,
{
    let ten = FloatType::from_f64(10.0).unwrap();
    let factor = FloatType::from_f64(0.025).unwrap();
    ten.powf(gain_db * factor)
}

#[test]
fn interleave_and_deinterleave() {
    let interleaved = vec![1., 2., 3., 1., 2., 3., 1., 2., 3.];
    let num_channels = 3;
    let mut result = vec![0.; 9];
    let expected = vec![1., 1., 1., 2., 2., 2., 3., 3., 3.];
    deinterleave(&interleaved, &mut result, num_channels);
    assert_eq!(result, expected);

    let mut new_result = vec![0.; 9];
    interleave(&result, &mut new_result, num_channels);
    assert_eq!(new_result, interleaved);
}
