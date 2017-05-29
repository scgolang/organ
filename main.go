package main

import (
	"log"
	"math"
	"strings"
	"time"

	"github.com/scgolang/midi"
	"github.com/scgolang/sc"
)

func main() {
	client, err := sc.NewClient("udp", "0.0.0.0:0", "127.0.0.1:57120", 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	group, err := client.AddDefaultGroup()
	if err != nil {
		log.Fatal(err)
	}
	if err := client.SendDef(def); err != nil {
		log.Fatal(err)
	}
	devices, err := midi.Devices()
	if err != nil {
		log.Fatal(err)
	}
	var keystation *midi.Device
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), "keystation") {
			keystation = d
			break
		}
	}
	if keystation == nil {
		log.Fatal("no keystation detected")
	}
	if err := keystation.Open(); err != nil {
		log.Fatal(err)
	}
	packets, err := keystation.Packets()
	if err != nil {
		log.Fatal(err)
	}
	var synths [127]*sc.Synth

	for pkt := range packets {
		if pkt.Err != nil {
			log.Fatal(pkt.Err)
		}
		gate := float32(0)
		if pkt.Data[2] > 0 {
			gate = float32(1)
		} else {
			if err := synths[pkt.Data[1]].Set(map[string]float32{"gate": gate}); err != nil {
				log.Fatal(err)
			}
			continue
		}
		ctls := map[string]float32{
			"amp":         float32(pkt.Data[2]) / float32(127),
			"fundamental": sc.Midicps(float32(pkt.Data[1])),
			"gate":        gate,
		}
		id := client.NextSynthID()
		synth, err := group.Synth("organ", id, sc.AddToTail, ctls)
		if err != nil {
			log.Fatal(err)
		}
		synths[pkt.Data[1]] = synth
	}
}

var def = sc.NewSynthdef("organ", func(params sc.Params) sc.Ugen {
	const numPartials = 5

	var (
		amp         = params.Add("amp", 0.9)
		fundamental = params.Add("fundamental", 440)
		gate        = params.Add("gate", 1)
		voices      = getVoices(numPartials, fundamental, amp)
		sig         = sc.Mix(sc.AR, voices).Mul(sc.EnvGen{
			Done: sc.FreeEnclosing,
			Env: sc.EnvADSR{
				A: sc.C(0.01),
				D: sc.C(1),
				S: sc.C(1),
				R: sc.C(0.1),
			},
			Gate: gate,
		}.Rate(sc.KR))
	)
	return sc.Out{
		Bus:      sc.C(0),
		Channels: sc.Multi(sig, sig),
	}.Rate(sc.AR)
})

func getVoices(n int, fundamental, amp sc.Input) []sc.Input {
	voices := make([]sc.Input, n)
	for i := range voices {
		voiceAmp := sc.C(float32(1) / float32(math.Pow(2, float64(i+1)))).Mul(amp)
		voices[i] = sc.SinOsc{
			Freq: fundamental.Mul(sc.C(float32(1) / float32(math.Pow(2, float64(i))))),
		}.Rate(sc.AR).Mul(voiceAmp)
	}
	return voices
}
