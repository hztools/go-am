// {{{ Copyright (c) Paul R. Tagliamonte <paul@k3xec.com>, 2022
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE. }}}

package am

import (
	"math/cmplx"

	"hz.tools/am/internal"
	"hz.tools/rf"
	"hz.tools/sdr"
	"hz.tools/sdr/fft"
	"hz.tools/sdr/stream"
)

// Reader will allow for the reading of AM demodulated audio samples from
// an IQ stream.
type Reader interface {
	Read([]float32) (int, error)
}

var (
	// BroadcastDeviation is the max deviation for AM Broadcast
	// which is 5 KHz (10 KHz bandwidth).
	BroadcastDeviation rf.Hz = 5 * rf.KHz
)

// DemodulatorConfig will define how the demodulator should decode audio from
// the iq data.
type DemodulatorConfig struct {
	// Center frequency of the signal in the IQ data.
	CenterFrequency rf.Hz

	// Deviation is the maximum difference between modulated and carrier
	// frequencies. This is half of the total bandwidth.
	Deviation rf.Hz

	// Downsample will define rate to downsample the samples to bring it to
	// a sensible audio sample rate.
	Downsample uint

	// Planner will be used to perform the FFTs used to filter the FM signal.
	Planner fft.Planner
}

// Demodulator contains info about
type Demodulator struct {
	reader sdr.Reader
	config DemodulatorConfig
}

// Reader will return the underlying reader (TODO: Remove this)
func (d Demodulator) Reader() sdr.Reader {
	return d.reader
}

// SampleRate will return the *audio* sample rate.
func (d Demodulator) SampleRate() uint {
	return uint(d.reader.SampleRate())
}

// Read will (partially?) fill the buffer with audio samples.
func (d Demodulator) Read(audio []float32) (int, error) {
	buf := make(sdr.SamplesC64, len(audio))
	i, err := sdr.ReadFull(d.reader, buf)
	if err != nil {
		return 0, err
	}
	buf = buf[:i]

	for i := 0; i < len(buf); i++ {
		audio[i] = float32(cmplx.Abs(complex128(buf[i])))
	}
	return len(buf), nil
}

// Demodulate will create a new Demodulator, to read FM audio
// from an IQ stream.
func Demodulate(reader sdr.Reader, cfg DemodulatorConfig) (*Demodulator, error) {
	var err error

	switch reader.SampleFormat() {
	case sdr.SampleFormatC64:
	default:
		return nil, sdr.ErrSampleFormatMismatch
	}

	filter := make([]complex64, 1024*32)
	if err := internal.Filter(
		filter,
		reader.SampleRate(),
		fft.ZeroFirst,
		cfg.CenterFrequency,
		cfg.Deviation,
	); err != nil {
		return nil, err
	}

	reader, err = stream.ConvolutionReader(reader, cfg.Planner, filter)
	if err != nil {
		return nil, err
	}

	reader, err = stream.DownsampleReader(reader, cfg.Downsample)
	if err != nil {
		return nil, err
	}

	return &Demodulator{
		reader: reader,
		config: cfg,
	}, nil
}

// vim: foldmethod=marker
