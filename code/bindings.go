package main

import (
	"strings"
	"fmt"
    "strconv"
)

// @TODO Should we add 'scroll sensitivity'?
// @TODO When a gamepad button is held down, the mouse or keyboard input should fire continuously.

// Thumbstick value scaling
const (
    Constant = iota
	Linear
	Squared
	Cubed
)

type Bindings struct {
    Bindings            map[*GamepadInput]*MouseOrKeyboardInput
    ThumbstickScaling   int
    MouseSensitivity    float64
    ThumbstickDeadZone  float64
    TriggerThreshold    float64
}

func NewBindings() Bindings {
    b := Bindings{}
	b.Bindings = map[*GamepadInput]*MouseOrKeyboardInput{}
    b.ThumbstickScaling  = Linear
    b.MouseSensitivity   = 1.0
    b.ThumbstickDeadZone = DefaultThumbstickDeadZone
    b.TriggerThreshold   = DefaultTriggerThreshold
    return b
}

func ParseBindings(contents string) (Bindings, error) {

    reportError := func(lineNumber int, errorString string) error {
        return fmt.Errorf("Error on line %d: %s", lineNumber, errorString)
    }

    bindings := NewBindings()
    contents = strings.Replace(contents, "\r\n", "\n", -1) // Remove Windows carriage return
    lines := strings.Split(contents, "\n")

    for i, line := range(lines) {
		line = strings.SplitN(line, "#", 2)[0] // Split the line at the first occurence of '#' and get the left side
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
        line = strings.ToUpper(line)
        line = strings.Replace(line, "_", "", -1) // Remove underscores
		split := strings.Split(line, "=")
		if len(split) != 2 {
            return bindings, reportError(i+1, "expected exactly one equals sign.")
		}

        lhs := strings.TrimSpace(split[0])
        rhs := strings.TrimSpace(split[1])
        if len(lhs) == 0 {
            return bindings, reportError(i+1, "empty left hand side.")
        }
        if len(rhs) == 0 {
            return bindings, reportError(i+1, "empty right hand side.")
        }

        inputError := parseInput(&bindings, lhs, rhs)
        var constantError error = nil
        if inputError != nil {
            constantError = parseConstant(&bindings, lhs, rhs)
        }

        if inputError != nil && constantError != nil {
            // @TODO: Make use of the error messages.
            // We don't know if the user attempted to bind an input or assign a constant so we cannot give a better error message.
            return bindings, reportError(i+1, "could not convert this line into a binding or constant assignment.")
        }
    }

    // Apply mouse sensitivity.
    for _,v := range(bindings.Bindings) {
        if v.IsMouseMove {
            v.X = LONG(float64(v.X) * bindings.MouseSensitivity)
            v.Y = LONG(float64(v.Y) * bindings.MouseSensitivity)
        }
    }

	return bindings, nil
}

// @TODO Support RHS keycodes like 0x50.
func parseInput(bindings *Bindings, lhs, rhs string) (error) {
    // Gamepad input
    button, found := StringToGamepadButton[lhs]
    var gpInput GamepadInput
    if found {
        gpInput = NewGamepadButtonInput(WORD(button))
    } else if lhs == "LTRIGGER" || lhs == "LEFTTRIGGER" {
        gpInput = NewGamepadTriggerInput(true)
    } else if lhs == "RTRIGGER" || lhs == "RIGHTTRIGGER" {
        gpInput = NewGamepadTriggerInput(false)
    } else if lhs == "LTHUMBX" || lhs ==  "LEFTTHUMBX" ||
              lhs == "LEFTTHUMBSTICKX" || lhs == "LEFTSTICKX" ||
              lhs == "LSTICKX" || lhs == "LTHUMBSTICKX" {
        gpInput = NewGamepadThumbstickInput(true, true)
    } else if lhs == "LTHUMBY" || lhs ==  "LEFTTHUMBY" ||
              lhs == "LEFTTHUMBSTICKY" || lhs == "LEFTSTICKY" ||
              lhs == "LSTICKY" || lhs == "LTHUMBSTICKY" {
        gpInput = NewGamepadThumbstickInput(true, false)
    } else if lhs == "RTHUMBX" || lhs ==  "RIGHTTHUMBX" ||
              lhs == "RIGHTTHUMBSTICKX" || lhs == "RIGHTSTICKX" ||
              lhs == "RSTICKX" || lhs == "RTHUMBSTICKX" {
        gpInput = NewGamepadThumbstickInput(false, true)
    } else if lhs == "RTHUMBY" || lhs ==  "RIGHTTHUMBY" ||
              lhs == "RIGHTTHUMBSTICKY" || lhs == "RIGHTSTICKY" ||
              lhs == "RSTICKY" || lhs == "RTHUMBSTICKY" {
        gpInput = NewGamepadThumbstickInput(false, false)
    } else {
        return fmt.Errorf("left hand side isn't a gamepad input.")
    }
    // It is guaranteed that gpInput is set at this point

    key, found := StringToKeyboardKey[rhs]
    var mkInput MouseOrKeyboardInput
    if found {
        // Keyboard input
        mkInput = NewKeyboardInput(DWORD(key))
    } else {
        // @TODO This logic probably works but seriously, we need to have the logic in this function in other functions for easier flow control and readability.
        /* {
            // Keyboard keycodes may be expressed as numbers.
            // Here we will attempt to parse rhs to an uint16 (WORD)
            if strings.HasPrefix(rhs, "0x") {
                key, err := strconv.ParseUint(rhs[2:], 16, 16)
                if err == nil {
                    mkInput = NewKeyboardInput(DWORD(key))
                    break
                }
            } else {
                key, err := strconv.ParseUint(rhs, 10, 16)
                if err == nil {
                    mkInput = NewKeyboardInput(DWORD(key))
                }
            }
        } */

        // Mouse input
        mouseButton, found := StringToMouseButton[rhs]
        if found {
            mkInput = NewMouseButtonInput(DWORD(mouseButton))
        } else {
            switch rhs {
                case "SCROLLDOWN":
                    // @TODO This should be negative, but that's outside the range of DWORD.
                    mkInput = NewScrollInput(WHEEL_DELTA)
                case "SCROLLUP":
                    mkInput = NewScrollInput(WHEEL_DELTA)
                case "MOUSEUP":
                    mkInput = NewMouseMoveInput(0, -1)
                case "MOUSEDOWN":
                    mkInput = NewMouseMoveInput(0, 1)
                case "MOUSELEFT":
                    mkInput = NewMouseMoveInput(-1, 0)
                case "MOUSERIGHT":
                    mkInput = NewMouseMoveInput(1, 0)
                case "MOUSEX":
                    panic("Unsupported") // @TODO
                case "MOUSEY":
                    panic("Unsupported") // @TODO
                default:
                    return fmt.Errorf("right hand side isn't a mouse or keyboard input.")
            }
        }
    }
    // It is guaranteed that mkInput is set at this point
    // @TODO What do we do if the key/value is already assigned?
    bindings.Bindings[&gpInput] = &mkInput
    return nil
}

func parseConstant(bindings *Bindings, lhs, rhs string) (error) {
    if lhs == "DEADZONE" {
        zone, err := strconv.ParseFloat(rhs, 64)
        if err == nil {
            bindings.ThumbstickDeadZone = zone
            return nil
        } else {
            return fmt.Errorf("right hand side isn't a number.")
        }
    } else if lhs == "THRESHOLD" {
        thresh, err := strconv.ParseFloat(rhs, 64)
        if err == nil {
            bindings.TriggerThreshold = thresh
            return nil
        } else {
            return fmt.Errorf("right hand side isn't a number.")
        }
    } else if lhs == "MOUSESENSITIVITY" {
        sens, err := strconv.ParseFloat(rhs, 64)
        if err == nil {
            bindings.MouseSensitivity = sens
            return nil
        } else {
            return fmt.Errorf("right hand side isn't a number.")
        }
    } else if lhs == "STICKSCALING" {
        var scaling int
        switch rhs {
            case "CONSTANT":
                scaling = Constant
            case "LINEAR":
                scaling = Linear
            case "SQUARED":
                scaling = Squared
            case "CUBED":
                scaling = Cubed
            default:
                scaling = -1
        }
        if scaling == -1 {
            return fmt.Errorf("unknown scaling mode. Please use \"linear\", \"squared\" or \"cubed\".")
        } else {
            bindings.ThumbstickScaling = scaling
            return nil
        }
    } else {
        return fmt.Errorf("left hand side is not a known constant.")
    }
    panic("Internal error: unexpected code path")
}
