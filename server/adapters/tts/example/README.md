# Eleven Labs Text-to-Speech Example

This example demonstrates how to use the Eleven Labs Text-to-Speech adapter in the Arunika project.

## Prerequisites

- Go 1.16 or higher
- An Eleven Labs API key (sign up at [elevenlabs.io](https://elevenlabs.io/))
- For audio playback: SoX, FFmpeg, or ALSA (depending on your operating system)

## Setup

1. Create a `.env` file in this directory:

```bash
touch .env
```

2. Add your Eleven Labs API key to the `.env` file:

```
ELEVEN_LABS_API_KEY=your_eleven_labs_api_key_here
```

Note: For a complete list of environment variables, see the main `.env.sample` file in the server root directory.

3. (Optional) Customize other settings in the `.env` file as needed.

## Running the Example

Execute the following command from this directory:

```bash
go run main.go
```

This will:
1. Convert the sample text to speech using Eleven Labs API
2. Save the audio to `example_output.pcm`
3. Attempt to play the audio automatically (if a compatible player is available)

## Playback Options

To see a list of available voices:

```bash
SHOW_VOICES=true go run main.go
```

To disable automatic playback:

```bash
NO_AUTOPLAY=true go run main.go
```

## Audio Format

The default output is PCM audio (24kHz, 16-bit signed, mono). To play this format:

### Using SoX (recommended)
```bash
play -t raw -r 24000 -e signed -b 16 -c 1 example_output.pcm
```

### Using FFplay
```bash
ffplay -f s16le -ar 24000 -ac 1 -nodisp -autoexit example_output.pcm
```

### Convert to WAV for easier playback
```bash
ffmpeg -f s16le -ar 24000 -ac 1 -i example_output.pcm example_output.wav
```
