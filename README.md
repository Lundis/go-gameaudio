# go-gameaudio

[![Go Reference](https://pkg.go.dev/badge/github.com/Lundis/go-gameaudio.svg)](https://pkg.go.dev/github.com/Lundis/go-gameaudio)
[![Build Status](https://github.com/Lundis/go-gameaudio/actions/workflows/test.yml/badge.svg)](https://github.com/Lundis/go-gameaudio/actions?query=workflow%3Atest)

An opinionated fork of [Oto](https://github.com/ebitengine/oto). 

I've stripped out io.Reader, instead using only []float32 - no conversions to deal with anywhere. 
You need to load your sounds into memory.

## Changes compared to Oto:
- Player was renamed to Sound.
- One sound can play multiple times simultaneously, without needing to create multiple instances of it.
- Looping sounds (with crossfade), for simple music or ambient setups
- Playing sounds with fade in. Randomize the fadein a tiny bit to make SFX sound less repetitive! 
- Sounds are tied to channels, controlling volume and pausing on the channel level, which is more in line with what you do in a game.
- Much less memory copying and conversions during playback due to always working on []float32 instead of []byte and io.Reader.

## Future plans:
- dynamic audio processing (for static processing, just do it on the audio data before creating the Sound)

#### Table of Contents:
- [go-gameaudio](#go-gameaudio)
  - [Usage](#usage)
  - [Platforms](#platforms)
  - [Prerequisite](#prerequisite)
    - [macOS](#macos)
    - [iOS](#ios)
    - [Linux](#linux)
    - [FreeBSD, OpenBSD](#freebsd-openbsd)
  - [Crosscompiling](#crosscompiling)


## Usage

The two main components are a `Context` and `Sound`. The context handles interactions with
the OS and audio drivers, and as such there can only be **one** context in your program.

From a context you can create any number of different players, where each player is given an `io.Reader` that
it reads bytes representing sounds from and plays.

```go
package main

import (
    "time"

    "github.com/Lundis/go-gameaudio/audio"
    "github.com/Lundis/go-gameaudio/loaders/wav"
)

const sampleRate = 44100

func main() {
	// audio files must have the expected sample rate, this library does not resample
    audioData, err := wav.LoadWav("loaders/wav/test_stereo.wav", sampleRate)
    if err != nil {
        panic(err)
    }

    // Prepare a context (this will use your default audio device) that will
    // play all our sounds. Its configuration can't be changed later.

    op := &audio.NewContextOptions{
        // Usually 44100 or 48000. Other values might cause distortions
		SampleRate: sampleRate,
        // Number of channels (aka locations) to play sounds from. Either 1 or 2.
        // 1 is mono sound, and 2 is stereo (most speakers are stereo). 
        ChannelCount: 2,
	    // audio device buffer size in time. audio devices may ignore this.
	    BufferSize: 10 * time.Millisecond,
    }

    // Remember that you should **not** create more than one context
    context, readyChan, err := audio.NewContext(op)
    if err != nil {
        panic("oto.NewContext failed: " + err.Error())
    }
    // It might take a bit for the hardware audio devices to be ready, so we wait on the channel.
    <-readyChan

    // Create a new 'player' that will handle our sound. Paused by default.
    player := context.NewSound(audioData, 1, audio.ChannelIdDefault)
    
    // Play starts playing the sound and returns without waiting for it (Play() is async).
    player.Play()

    // We can wait for the sound to finish playing using something like this
    for player.IsPlaying() {
        time.Sleep(time.Millisecond)
    }
}
```

See the examples folder for more examples.

## Platforms

- Windows (no Cgo required!)
- macOS (no Cgo required!)
- Linux
- FreeBSD
- OpenBSD
- Android
- iOS
- WebAssembly
- Nintendo Switch
- Xbox

## Prerequisite

On some platforms you will need a C/C++ compiler in your path that Go can use.

- iOS: On newer macOS versions type `clang` on your terminal and a dialog with installation instructions will appear if you don't have it
  - If you get an error with clang use xcode instead `xcode-select --install`
- Linux and other Unix systems: Should be installed by default, but if not try [GCC](https://gcc.gnu.org/) or [Clang](https://releases.llvm.org/download.html)

### macOS

This requires `AudioToolbox.framework`, but this is automatically linked.

### iOS

This requires these frameworks:

- `AVFoundation.framework`
- `AudioToolbox.framework`

Add them to "Linked Frameworks and Libraries" on your Xcode project.

### Linux

ALSA is required. On Ubuntu or Debian, run this command:

```sh
apt install libasound2-dev gcc pkg-config
```

On RedHat-based linux distributions, run:

```sh
dnf install alsa-lib-devel
```

In most cases this command must be run by root user or through `sudo` command.

### FreeBSD, OpenBSD

BSD systems are not tested well. If ALSA works, this should work.

## Crosscompiling

Crosscompiling to macOS or Windows is as easy as setting `GOOS=darwin` or `GOOS=windows`, respectively.

To crosscompile for other platforms, make sure the libraries for the target architecture are installed, and set 
`CGO_ENABLED=1` as Go disables [Cgo](https://golang.org/cmd/cgo/#hdr-Using_cgo_with_the_go_command) on crosscompiles by default.
