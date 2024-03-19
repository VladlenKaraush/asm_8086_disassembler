package main

import (
	"fmt"
	"os"
)

type command struct {
	cmd    string
	source string
	dest   string
}

func (c *command) Str() string {
	return fmt.Sprintf("%s %s, %s", c.cmd, c.dest, c.source)
}

var regTable = map[byte]string{
	0b0000: "al", 0b0001: "ax",
	0b0010: "cl", 0b0011: "cx",
	0b0100: "dl", 0b0101: "dx",
	0b0110: "bl", 0b0111: "bx",
	0b1000: "ah", 0b1001: "sp",
	0b1010: "ch", 0b1011: "bp",
	0b1100: "dh", 0b1101: "si",
	0b1110: "bh", 0b1111: "di",
}

func parseCommand(bytes []byte) command {
	opcode := bytes[0] >> 2
	word := bytes[0] & 1
	direction := (bytes[0] >> 1) & 1
	rm := bytes[1] & 7
	reg := (bytes[1] >> 3) & 7
	mod := bytes[1] >> 6

	cmd := command{}
	if opcode == 0b100010 {
		cmd.cmd = "mov"
	} else {
		panic("operation not supported")
	}

	regStr := regTable[reg<<1|word]

	if mod == 0b11 {
		rmStr := regTable[rm<<1|word]
		if direction == 0b1 {
			cmd.dest = regStr
			cmd.source = rmStr
		} else {
			cmd.dest = rmStr
			cmd.source = regStr
		}
	} else {
		panic("memory mode is not supported")
	}
	return cmd
}

func main() {
	filename := os.Args[1]
	bytes, _ := os.ReadFile(filename)

	var commands []command
	for i := 0; i < len(bytes); i += 2 {
		commands = append(commands, parseCommand(bytes[i:i+2]))
	}
	for _, cmd := range commands {
		fmt.Printf("%s\n", cmd.Str())
	}
}
