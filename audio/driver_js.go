// Copyright 2021 The Oto Authors
// Copyright 2025 Lundis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package audio

import (
	"errors"
	"fmt"
	"runtime"
	"syscall/js"
	"unsafe"
)

var jsContext struct {
	audioContext            js.Value
	scriptProcessor         js.Value
	scriptProcessorCallback js.Func
	ready                   bool
}

func newContext(bufferSizeInBytes int) (chan struct{}, error) {
	ready := make(chan struct{})

	class := js.Global().Get("AudioContext")
	if !class.Truthy() {
		class = js.Global().Get("webkitAudioContext")
	}
	if !class.Truthy() {
		return nil, errors.New("oto: AudioContext or webkitAudioContext was not found")
	}
	options := js.Global().Get("Object").New()
	options.Set("sampleRate", mux.sampleRate)

	jsContext.audioContext = class.New(options)

	if bufferSizeInBytes == 0 {
		// 4096 was not great at least on Safari 15.
		bufferSizeInBytes = 8192 * ChannelCount
	}

	buf32 := make([]float32, bufferSizeInBytes/4)

	if w := jsContext.audioContext.Get("audioWorklet"); w.Truthy() {
		script := fmt.Sprintf(`
class OtoWorkletProcessor extends AudioWorkletProcessor {
	constructor() {
		super();
		this.bufferSize_ = %[1]d;
		this.channelCount_ = %[2]d;
		this.buf_ = new Float32Array();
		this.waitRecv_ = false;

		// Receive data from the main thread.
		this.port.onmessage = (event) => {
			const buf = event.data;
			const newBuf = new Float32Array(this.buf_.length + buf.length);
			newBuf.set(this.buf_);
			newBuf.set(buf, this.buf_.length);
			this.buf_ = newBuf;
			this.waitRecv_ = false;
		}
	}

	process(inputs, outputs, parameters) {
		const output = outputs[0];

		// If the buffer is too short, request more data and return silence.
		if (this.buf_.length < output[0].length*this.channelCount_) {
			if (!this.waitRecv_) {
				this.waitRecv_ = true;
				this.port.postMessage(null);
			}
			for (let i = 0; i < output.length; i++) {
				output[i].fill(0);
			}
			return true;
		}

		// If the buffer is short, request more data.
		if (this.buf_.length < this.bufferSize_*this.channelCount_ / 2 && !this.waitRecv_) {
			this.waitRecv_ = true;
			this.port.postMessage(null);
		}

		for (let i = 0; i < this.channelCount_; i++) {
			for (let j = 0; j < output[i].length; j++) {
				output[i][j] = this.buf_[j*this.channelCount_+i];
			}
		}
		this.buf_ = this.buf_.slice(output[0].length*this.channelCount_);
		return true;
	}
}
registerProcessor('oto-worklet-processor', OtoWorkletProcessor);
`, bufferSizeInBytes/4/ChannelCount, ChannelCount)
		w.Call("addModule", newScriptURL(script)).Call("then", js.FuncOf(func(this js.Value, arguments []js.Value) any {
			node := js.Global().Get("AudioWorkletNode").New(jsContext.audioContext, "oto-worklet-processor", map[string]any{
				"outputChannelCount": []any{ChannelCount},
			})
			port := node.Get("port")
			// When the worklet processor requests more data, send the request to the worklet.
			port.Set("onmessage", js.FuncOf(func(this js.Value, arguments []js.Value) any {
				mux.ReadFloat32s(buf32)
				buf := float32SliceToTypedArray(buf32)
				port.Call("postMessage", buf, map[string]any{
					"transfer": []any{buf.Get("buffer")},
				})
				return nil
			}))
			node.Call("connect", jsContext.audioContext.Get("destination"))
			return nil
		}))
	} else {
		// Use ScriptProcessorNode if AudioWorklet is not available.

		chBuf32 := make([][]float32, ChannelCount)
		for i := range chBuf32 {
			chBuf32[i] = make([]float32, len(buf32)/ChannelCount)
		}

		sp := jsContext.audioContext.Call("createScriptProcessor", bufferSizeInBytes/4/ChannelCount, 0, ChannelCount)
		f := js.FuncOf(func(this js.Value, arguments []js.Value) any {
			mux.ReadFloat32s(buf32)
			for i := 0; i < ChannelCount; i++ {
				for j := range chBuf32[i] {
					chBuf32[i][j] = buf32[j*ChannelCount+i]
				}
			}

			buf := arguments[0].Get("outputBuffer")
			if buf.Get("copyToChannel").Truthy() {
				for i := 0; i < ChannelCount; i++ {
					buf.Call("copyToChannel", float32SliceToTypedArray(chBuf32[i]), i, 0)
				}
			} else {
				// copyToChannel is not defined on Safari 11.
				for i := 0; i < ChannelCount; i++ {
					buf.Call("getChannelData", i).Call("set", float32SliceToTypedArray(chBuf32[i]))
				}
			}
			return nil
		})
		sp.Call("addEventListener", "audioprocess", f)
		jsContext.scriptProcessor = sp
		jsContext.scriptProcessorCallback = f
		sp.Call("connect", jsContext.audioContext.Get("destination"))
	}

	// Browsers require user interaction to start the audio.
	// https://developers.google.com/web/updates/2017/09/autoplay-policy-changes#webaudio

	events := []string{"touchend", "keyup", "mouseup"}

	var onEventFired js.Func
	var onResumeSuccess js.Func
	onResumeSuccess = js.FuncOf(func(this js.Value, arguments []js.Value) any {
		jsContext.ready = true
		close(ready)
		for _, event := range events {
			js.Global().Get("document").Call("removeEventListener", event, onEventFired)
		}
		onEventFired.Release()
		onResumeSuccess.Release()
		return nil
	})
	onEventFired = js.FuncOf(func(this js.Value, arguments []js.Value) any {
		if !jsContext.ready {
			jsContext.audioContext.Call("resume").Call("then", onResumeSuccess)
		}
		return nil
	})
	for _, event := range events {
		js.Global().Get("document").Call("addEventListener", event, onEventFired)
	}

	return ready, nil
}

func float32SliceToTypedArray(s []float32) js.Value {
	bs := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)
	a := js.Global().Get("Uint8Array").New(len(bs))
	js.CopyBytesToJS(a, bs)
	runtime.KeepAlive(s)
	buf := a.Get("buffer")
	return js.Global().Get("Float32Array").New(buf, a.Get("byteOffset"), a.Get("byteLength").Int()/4)
}

func newScriptURL(script string) js.Value {
	blob := js.Global().Get("Blob").New([]any{script}, map[string]any{"type": "text/javascript"})
	return js.Global().Get("URL").Call("createObjectURL", blob)
}
