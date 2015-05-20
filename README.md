# bitratemon
Monitor bitrate of given video stream to see how it swings.

## Prerequisites
- This application invokes an `ffprobe` command with `-show_entries frame -print_format json` option. Please make sure that you have this command available.
