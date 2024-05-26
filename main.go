package main

import (
	. "asm/model"
	"asm/parser"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type state struct {
	regs     map[string]int
	flags    map[string]bool
	memory   []byte
	commands []Command
	ip       int // instruction pointer
}

func initState(cmds []Command) state {
	state := state{
		regs: map[string]int{
			"ax": 0, "cx": 0, "dx": 0, "bx": 0, "sp": 0, "bp": 0, "si": 0, "di": 0,
		},
		flags: map[string]bool{
			"s": false, "z": false,
		},
		memory:   make([]byte, 65536),
		ip:       0,
		commands: cmds,
	}
	return state
}

func setFlags(val int, flags map[string]bool) {
	switch true {
	case val == 0:
		flags["z"] = true
		flags["s"] = false
		break
	case val < 0:
		flags["z"] = false
		flags["s"] = true
	case val > 0:
		flags["z"] = false
		flags["s"] = true
	}
}

func getMemAddr(addr string) (int, error) {
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		val, err := strconv.Atoi(strings.Trim(addr, "[] "))
		if err != nil {
			return 0, err
		}
		return val, nil
	} else {
		return 0, errors.New("addr is not memory location")
	}
}

func (s state) exec(cmd Command) int {
	// fmt.Printf("executing command %v\n", cmd)
	var sourceVal, newVal, jump int
	var sourceValFromMem []byte = nil

	jump = 0
	if regVal, ok := s.regs[cmd.Source]; ok {
		sourceVal = regVal
	} else if sourceMemAddr, err := getMemAddr(cmd.Source); err == nil {
		sourceValFromMem = make([]byte, s.memory[sourceMemAddr])
	} else {
		immediateVal, _ := strconv.Atoi(cmd.Source)
		sourceVal = immediateVal
	}
	switch cmd.Cmd {
	case "mov":
		newVal = sourceVal
		setFlags(newVal, s.flags)
	case "add":
		newVal = s.regs[cmd.Dest] + sourceVal
		setFlags(newVal, s.flags)
	case "cmp":
		newVal = s.regs[cmd.Dest]
		setFlags(s.regs[cmd.Dest]-sourceVal, s.flags)
	case "sub":
		newVal = s.regs[cmd.Dest] - sourceVal
		setFlags(newVal, s.flags)
	case "jnz":
		if s.flags["z"] == false {
			jumpOffset, err := strconv.Atoi(cmd.Dest)
			if err != nil {
				panic(err)
			}
			jump = jumpOffset
		}
	}
	if _, ok := s.regs[cmd.Dest]; ok {
		fmt.Printf("executing cmd %v, reg %s: %d -> %d \n", cmd, cmd.Dest, s.regs[cmd.Dest], newVal)
		s.regs[cmd.Dest] = newVal
	} else if destMemAddr, err := getMemAddr(cmd.Dest); err == nil {
		if sourceValFromMem != nil {
			s.memory[destMemAddr] = sourceValFromMem[0]
		} else {
			s.memory[destMemAddr] = byte(newVal)
		}
	} else {
		fmt.Printf("executing cmd %v, jump offset: %d \n", cmd, jump)
	}
	return jump
}

func (s *state) executeCommands() {
	for s.ip < len(s.commands) {
		cmd := s.commands[s.ip]
		offset := s.exec(cmd)
		// offset := cmd.exec(s.regs, s.flags)
		for offset < 0 {
			cmd := s.commands[s.ip]
			offset += cmd.Len
			s.ip -= 1
		}
		s.ip += 1
	}
	fmt.Printf("final state: regs = %v, flags = %v\n", s.regs, s.flags)
}

func main() {

	filename := os.Args[1]
	var exec string

	if len(os.Args) > 2 {
		exec = os.Args[2]
	}

	bytes, _ := os.ReadFile(filename)
	fmt.Printf("input bytes = %b\n", bytes)

	commands := parser.ParseBytes(bytes)
	for _, cmd := range commands {
		fmt.Printf("%s\n", cmd.Str())
	}

	if exec == "--exec" {
		state := initState(commands)
		state.executeCommands()
	}
}
