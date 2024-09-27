package main

import (
	"fmt"
	"image/color"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/kettek/simple-embedded-music-mixer/assets"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type Song struct {
	player         *audio.Player
	slider         *widget.Slider
	button         *widget.Button
	fadingDuration time.Duration
	length         time.Duration
	volume         float64
}

var audioContext *audio.Context

var songs map[string]Song = make(map[string]Song)

func musicMain() {
	for {
		<-time.After(50 * time.Millisecond)
		for n, s := range songs {
			/*if s.player.IsPlaying() {
				s.slider.SetValue(float64(s.player.Position()) / float64(s.length) * 100)
				s.slider.Refresh()
			}*/
			if s.fadingDuration > 0 {
				s.fadingDuration -= 100 * time.Millisecond
				if s.fadingDuration <= 0 {
					s.player.Pause()
					s.button.Icon = theme.MediaPlayIcon()
					s.button.Refresh()
					s.fadingDuration = 0
				} else {
					s.player.SetVolume(float64(s.fadingDuration) / float64(time.Second) * s.volume)
				}
			} else if s.fadingDuration < 0 {
				s.fadingDuration += 100 * time.Millisecond
				if s.fadingDuration >= 0 {
					s.player.SetVolume(s.volume)
					s.fadingDuration = 0
				} else {
					fmt.Println("fading", s.fadingDuration, 1.0-float64(-s.fadingDuration)/float64(time.Second)*s.volume)
					s.player.SetVolume((1.0 - float64(-s.fadingDuration)/float64(time.Second)) * s.volume)
				}
			}
			songs[n] = s
		}
	}
}

func stopSong(song string) {
	if s, ok := songs[song]; ok {
		s.fadingDuration = 1 * time.Second
		songs[song] = s
	}
}

func playSong(song string) {
	fmt.Println("play", song)
	for n, s := range songs {
		if song == n {
			s.fadingDuration = -1 * time.Second
		} else if s.player.IsPlaying() {
			fmt.Println("stopper", n)
			s.fadingDuration = 1 * time.Second
			s.button.Icon = theme.MediaPlayIcon()
			s.button.Refresh()
		}
		songs[n] = s
	}
	songs[song].player.Play()
	songs[song].button.Icon = theme.MediaPauseIcon()
	songs[song].button.Refresh()
}

func rewindSong(song string) {
	if s, ok := songs[song]; ok {
		s.button.Refresh()
		s.player.Rewind()
		s.slider.SetValue(0)
		s.slider.Refresh()
		s.fadingDuration = 0
		songs[song] = s
	}
}

func setVolume(song string, volume float64) {
	if s, ok := songs[song]; ok {
		if volume > 1.0 {
			volume = 1.0
		} else if volume < 0.0 {
			volume = 0.0
		}
		s.volume = volume
		s.player.SetVolume(volume)
		songs[song] = s
	}
}

type myTheme struct{}

// The color package has to be imported from "image/color".

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

var sizeModifier float32 = 1.5

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name) * sizeModifier
}

func main() {

	// Setup audio
	audioContext = audio.NewContext(44100)
	go musicMain()

	a := app.New()
	a.Settings().SetTheme(&myTheme{})
	w := a.NewWindow("kbwedding")

	middle := container.NewVBox()

	// Create our FS entries
	entries, err := assets.FS.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		// Trim suffix.
		name := e.Name()
		if len(name) > 4 {
			name = name[:len(name)-4]
		}

		// Get the song
		file, err := assets.FS.Open(e.Name())
		if err != nil {
			panic("opening mp3 failed: " + err.Error())
		}

		s, err := mp3.DecodeWithSampleRate(44100, file)
		if err != nil {
			panic(err)
		}

		player, err := audioContext.NewPlayer(s)
		if err != nil {
			panic(err)
		}

		label := widget.NewLabel(name)
		slider := widget.NewSlider(0, 100)
		volume := widget.NewEntry()

		volume.OnChanged = func(s string) {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return
			}
			setVolume(name, v)
		}
		volume.Text = "1.0"

		play := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
			if player.IsPlaying() {
				stopSong(name)
			} else {
				playSong(name)
			}
		})

		rewind := widget.NewButtonWithIcon("", theme.MediaFastRewindIcon(), func() {
			rewindSong(name)
		})

		songs[name] = Song{player, slider, play, 0, time.Second * time.Duration(s.Length()) / 4 / 44100, 1.0}

		outer := container.NewHBox(play, rewind, volume, label)
		container := container.NewVBox(outer)

		middle.Add(container)
	}

	sizeOptions := widget.NewSelect([]string{"Small", "Medium", "Large", "Larger"}, func(s string) {
		switch s {
		case "Small":
			sizeModifier = 1.0
		case "Medium":
			sizeModifier = 1.5
		case "Large":
			sizeModifier = 2.0
		case "Larger":
			sizeModifier = 2.5
		}
		a.Settings().SetTheme(&myTheme{})
	})
	sizeOptions.Selected = "Medium"

	sizeContainer := container.NewHBox(widget.NewLabel("UI Size"), sizeOptions)

	resetAll := widget.NewButton("Reset Songs", func() {
		for n, s := range songs {
			s.player.Pause()
			s.player.Rewind()
			s.fadingDuration = 0
			s.button.Icon = theme.MediaPlayIcon()
			s.button.Refresh()
			songs[n] = s
		}
	})

	top := container.NewVBox(sizeContainer, resetAll)

	content := container.NewBorder(top, nil, nil, nil, middle)
	w.SetContent(content)
	w.ShowAndRun()
}
