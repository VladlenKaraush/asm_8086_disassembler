package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
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

var sourceAddrTable = map[byte]string{
	0b000: "bx + si",
	0b001: "bx + di",
	0b010: "bp + si",
	0b011: "bp + di",
	0b100: "si",
	0b101: "di",
	0b110: "DIRECT ADDR",
	0b111: "bx",
}

func parseImmediateToReg(bytes []byte) (command, int) {
	word := bytes[0] >> 3 & 1
	reg := bytes[0] & 0b111
	dataStr := strconv.Itoa(int(bytes[1]))
	dataLen := 1
	if word == 1 {
		dataStr = strconv.Itoa(int(binary.LittleEndian.Uint16(bytes[1:3])))
		dataLen = 2
	}
	cmd := command{"mov", dataStr, regTable[reg<<1|word]}

	fmt.Printf("immediate to reg cmd='%s', word=%b, reg=%b len=%d\n", cmd.Str(), word, reg, dataLen)
	return cmd, dataLen
}

func parseRegToMem(word, direction, rm, reg byte) command {
	cmd := command{
		cmd:    "mov",
		dest:   regTable[reg<<1|word],
		source: "[" + sourceAddrTable[rm] + "]",
	}
	if direction == 0b0 {
		cmd.source, cmd.dest = cmd.dest, cmd.source
	}
	fmt.Printf("reg to/from effective address cmd='%s', word=%b, reg=%b\n", cmd.Str(), word, reg)
	return cmd
}

func parseRegToReg(word, direction, rm, reg byte) command {
	cmd := command{
		cmd:    "mov",
		dest:   regTable[rm<<1|word],
		source: regTable[reg<<1|word],
	}

	if direction == 0b1 {
		cmd.dest, cmd.source = cmd.source, cmd.dest
	}
	fmt.Printf("reg to reg cmd='%s', word=%b, reg=%b\n", cmd.Str(), word, reg)
	return cmd
}

func parseCommand(bytes []byte) (command, []byte) {

	fmt.Printf("parsing %08b bytes\n", bytes[:4])
	if bytes[0]>>4 == 0b1011 {
		// immediate to register case
		cmd, datalen := parseImmediateToReg(bytes)
		return cmd, bytes[datalen+1:]
	}

	opcode := bytes[0] >> 2
	// rm to/from register
	if opcode == 0b100010 {
		mod := bytes[1] >> 6
		word := bytes[0] & 1
		direction := (bytes[0] >> 1) & 1
		rm := bytes[1] & 0b111
		reg := (bytes[1] >> 3) & 0b111
		if mod == 0b11 {
			// reg to reg
			cmd := parseRegToReg(word, direction, rm, reg)
			return cmd, bytes[2:]
		}
		if mod == 0b00 {
			cmd := parseRegToMem(word, direction, rm, reg)
			return cmd, bytes[2:]
		}
	}
	panic("code not supported")
}

func main() {
	filename := os.Args[1]
	bytes, _ := os.ReadFile(filename)
	fmt.Printf("input bytes = %b\n", bytes)

	var commands []command

	for {
		cmd, restBytes := parseCommand(bytes)
		bytes = restBytes
		commands = append(commands, cmd)
		if restBytes == nil || len(restBytes) == 0 {
			break
		}
	}
	for _, cmd := range commands {
		fmt.Printf("%s\n", cmd.Str())
	}
}
