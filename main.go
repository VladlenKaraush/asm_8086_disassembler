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

var opcodeTable = map[byte]string{
	0b000000: "add",
	0b100000: "add",
	0b001010: "sub",
	0b001110: "cmp",
	0b100010: "mov",
	0b1011:   "mov",
}

var immediateOpTable = map[byte]string{
	0b000: "add",
	0b101: "sub",
	0b111: "cmp",
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
	0b110: "bp",
	0b111: "bx",
}

func parseImmediateAddToReg(bytes []byte) (command, int) {
	word := bytes[0] & 1
	s := bytes[0] >> 1 & 1
	rm := bytes[1] & 0b111
	mod := bytes[1] >> 6 & 0b111
	dataStr := strconv.Itoa(int(bytes[2]))
	dataLen := 1
	cmdBytes := bytes[1] >> 3 & 0b111
	// if word == 1 {
	// 	dataStr = strconv.Itoa(int(binary.LittleEndian.Uint16(bytes[2:4])))
	// 	dataLen = 2
	// }
	// add byte [bx], 34
	// [10000000 00000111 00100010 10000011] bytes
	// cmd := command{immediateOpTable[cmdBytes], dataStr, regTable[rm<<1|word]}

	// parsing [10000011 10000010 11101000 00000011] bytes
	// s = 1, w = 1, mod = 10, rm = 010
	// immediate add to reg cmd='add word [bp + si], 232', word=1, reg=10 len=1
	cmd := command{immediateOpTable[cmdBytes], dataStr, "[" + sourceAddrTable[rm] + "]"}
	var disp uint64
	switch mod {
	case 0b11:
		cmd.dest = sourceAddrTable[rm]
	case 0b00:
		cmd.dest = sourceAddrTable[rm]
	case 0b01:
		dataLen = 2
		disp = parseUintFromBytes(bytes[2:3])
		cmd.dest = "[" + sourceAddrTable[rm] + " + " + strconv.FormatUint(disp, 10) + "]"
		data := parseUintFromBytes(bytes[3:4])
		if word == 0b1 {
			cmd.dest = "word " + cmd.dest
			if s == 0b0 {
				data = parseUintFromBytes([]byte{bytes[4], bytes[3]})
				dataLen = 3
			}
		}
		cmd.source = strconv.FormatUint(data, 10)
	case 0b10:
		disp = parseUintFromBytes([]byte{bytes[3], bytes[2]})
		cmd.dest = "[" + sourceAddrTable[rm] + " + " + strconv.FormatUint(disp, 10) + "]"
		data := parseUintFromBytes(bytes[4:5])
		dataLen = 3
		if word == 0b1 {
			cmd.dest = "word " + cmd.dest
			if s == 0b0 {
				data = parseUintFromBytes([]byte{bytes[5], bytes[4]})
				dataLen = 4
			}
		}
		cmd.source = strconv.FormatUint(data, 10)
	}
	// if mod == 0b11 {
	// } else {
	// 	if word == 0b1 {
	// 		cmd.dest = "word " + cmd.dest
	// 	} else {
	// 		cmd.dest = "byte " + cmd.dest
	// 	}
	// }

	fmt.Printf("immediate add to reg cmd='%s', word=%b, reg=%b len=%d\n", cmd.Str(), word, rm, dataLen)
	return cmd, dataLen
}

func parseUintFromBytes(b []byte) uint64 {
	fmt.Println("parsing disp from bytes: ", b)
	var displacement uint64
	for _, byte := range b {
		displacement = displacement<<8 | uint64(byte)
	}
	return displacement
}

func parseImmediateToReg(opcode byte, bytes []byte) (command, int) {
	word := bytes[0] >> 3 & 1
	reg := bytes[0] & 0b111
	dataStr := strconv.Itoa(int(bytes[1]))
	dataLen := 1
	if word == 1 {
		dataStr = strconv.FormatUint(uint64(binary.LittleEndian.Uint16(bytes[1:3])), 10)
		dataLen = 2
	}
	cmd := command{opcodeTable[opcode], dataStr, regTable[reg<<1|word]}

	fmt.Printf("immediate to reg cmd='%s', word=%b, reg=%b len=%d\n", cmd.Str(), word, reg, dataLen)
	return cmd, dataLen
}

func parseRegToMem(opcode, word, direction, rm, reg byte) command {
	cmd := command{
		cmd:    opcodeTable[opcode],
		dest:   regTable[reg<<1|word],
		source: "[" + sourceAddrTable[rm] + "]",
	}
	if direction == 0b0 {
		cmd.source, cmd.dest = cmd.dest, cmd.source
	}
	fmt.Printf("reg to/from effective address cmd='%s', word=%b, reg=%b opcode=%06b\n", cmd.Str(), word, reg, opcode)
	return cmd
}

func parseRegToMemAndDisp(opcode, word, direction, rm, reg byte, disp []byte) command {
	dispInt := int(disp[0])
	if len(disp) == 2 {
		dispInt = int(binary.LittleEndian.Uint16(disp))
	}

	cmd := command{
		cmd:    opcodeTable[opcode],
		dest:   regTable[reg<<1|word],
		source: fmt.Sprintf("[%s + %d]", sourceAddrTable[rm], dispInt),
	}

	if direction == 0b0 {
		cmd.source, cmd.dest = cmd.dest, cmd.source
	}
	fmt.Printf("reg to/from effective address cmd='%s', word=%b, reg=%b\n", cmd.Str(), word, reg)
	return cmd
}

func parseRegToReg(opcode, word, direction, rm, reg byte) command {
	cmd := command{
		cmd:    opcodeTable[opcode],
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

	fmt.Printf("parsing %08b bytes\n", bytes[:6])
	if bytes[0]>>4 == 0b1011 {
		// immediate to register case
		cmd, datalen := parseImmediateToReg(bytes[0]>>4, bytes)
		return cmd, bytes[datalen+1:]
	}
	if bytes[0]>>2 == 0b100000 {
		cmd, datalen := parseImmediateAddToReg(bytes)
		return cmd, bytes[datalen+2:]
	}

	opcode := bytes[0] >> 2
	// mov rm to/from register
	mod := bytes[1] >> 6
	word := bytes[0] & 1
	direction := (bytes[0] >> 1) & 1
	rm := bytes[1] & 0b111
	reg := (bytes[1] >> 3) & 0b111
	if mod == 0b11 {
		// reg to reg
		cmd := parseRegToReg(opcode, word, direction, rm, reg)
		return cmd, bytes[2:]
	}
	if mod == 0b00 {
		cmd := parseRegToMem(opcode, word, direction, rm, reg)
		return cmd, bytes[2:]
	}
	if mod == 0b01 {
		// to reg from mem + 8 bit displacement
		cmd := parseRegToMemAndDisp(opcode, word, direction, rm, reg, bytes[2:3])
		return cmd, bytes[3:]
	}
	if mod == 0b10 {
		// to reg from mem + 16 bit displacement
		cmd := parseRegToMemAndDisp(opcode, word, direction, rm, reg, bytes[2:4])
		return cmd, bytes[4:]
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
		fmt.Println()
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
