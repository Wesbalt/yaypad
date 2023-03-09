package main

import (
	"time"
	"fmt"
	"io/ioutil"
	"runtime"
)

// @TODO Add support for hotloading.

func main() {
	if runtime.GOOS != "windows" {
		fmt.Println("Yaypad is only supported on Windows. Exiting.")
		return
	}
    // @TODO Take the path as a command-line argument!
    path := "bindings.yay"
	bytes, err := ioutil.ReadFile(path)
	panicIfNotNil(err)
    binds, err := ParseBindings(string(bytes))
	panicIfNotNil(err)

	GamepadConnectedCallback = func(userIndex int) {
		fmt.Println("gamepad connected")
	}
    GamepadDisconnectedCallback = func(userIndex int) {
		fmt.Println("gamepad disconnected")
	}
	// @TODO Using the arrow keys right now works as inteded in VSCode,
	// but test the behaviour in games. Is the input spammed or fired only once?
	/* GamepadPollCallback = func(userIndex int, state XInputState) {
		for in, out := range(binds.Bindings) {
			if state.InputValueBool(*in) {
				out.Send()
			}
		}
	} */
    GamepadInputCallback = func(userIndex int, state XInputState) {
		for in, out := range(binds.Bindings) {
			if state.InputValueBool(*in) {
				out.Send()
			}
		}
	}
	go PollGamepad(0)

	for {
		time.Sleep(time.Second) // Leaving this out causes the program to freeze after some time.
	}
}

func noop(any interface{}) {
	// ...
}

func panicIfNotNil(err error) {
	if (err != nil) {
		panic(err)
	}
}
