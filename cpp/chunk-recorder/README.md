Simple tool, that reads audio samples from an alsa device and writes it into a fifo.

The tool also detects if there is silence or actual audio data in the stream and publishes it
using avahi.

Besides writing into the fifo, it can also save data on a block device.
