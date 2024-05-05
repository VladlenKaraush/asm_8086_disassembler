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

var jumpTable = map[byte]string{
	0b01110101: "jnz",
	0b01110100: "je",
	0b01111100: "jl",
	0b01111110: "jle",
	0b01110010: "jb",
	0b01110110: "jbe",
	0b01111010: "jp",
	0b01110000: "jo",
	0b01111000: "js",
	// 0b01110101: "jne",
	0b01111101: "jnl",
	0b01111111: "jg",
	0b01110011: "jnb",
	0b01110111: "ja",
	0b01111011: "jnp",
	0b01110001: "jno",
	0b01111001: "jns",
	0b11100010: "loop",
	0b11100001: "loopz",
	0b11100000: "loopnz",
	0b11100011: "jcxz",
}

func parseJump(jmpName string, value byte) command {
	cmd := command{
		cmd:    jmpName,
		dest:   strconv.Itoa(int(int8(value))),
		source: "",
	}
	return cmd
}

func parseImmediateAddToReg(bytes []byte) (command, int) {
	word := bytes[0] & 1
	s := bytes[0] >> 1 & 1
	rm := bytes[1] & 0b111
	mod := bytes[1] >> 6 & 0b111
	dataStr := strconv.Itoa(int(bytes[2]))
	dataLen := 1
	cmdBytes := bytes[1] >> 3 & 0b111
	isWord, isByte := false, false
	cmd := command{immediateOpTable[cmdBytes], dataStr, sourceAddrTable[rm]}
	var disp uint64
	switch mod {
	case 0b11:
		cmd.dest = regTable[rm<<1|word]
	case 0b00:
		if rm == 0b110 {
			// direct address
			dataLen = 2
			disp = parseUintFromBytes(bytes[2:3])
			cmd.dest = strconv.FormatUint(disp, 10)
			data := parseUintFromBytes(bytes[3:4])
			if word == 0b1 {
				disp = parseUintFromBytes([]byte{bytes[3], bytes[2]})
				fmt.Printf("direct address disp = %d\n", disp)
				cmd.dest = strconv.FormatUint(disp, 10)
				isWord = true
				data = parseUintFromBytes(bytes[4:5])
				dataLen = 3
				if s == 0b0 {
					data = parseUintFromBytes([]byte{bytes[4], bytes[5]})
					dataLen = 4
				}
			} else {
				isByte = true
			}
			cmd.source = strconv.FormatUint(data, 10)
			// data = parseUintFromBytes(bytes[2:3])
		} else {
			cmd.dest = sourceAddrTable[rm]
		}
	case 0b01:
		dataLen = 2
		disp = parseUintFromBytes(bytes[2:3])
		cmd.dest = sourceAddrTable[rm] + " + " + strconv.FormatUint(disp, 10)
		data := parseUintFromBytes(bytes[3:4])
		if word == 0b1 {
			isWord = true
			if s == 0b0 {
				data = parseUintFromBytes([]byte{bytes[4], bytes[3]})
				dataLen = 3
			}
		} else {
			isByte = true
		}
		cmd.source = strconv.FormatUint(data, 10)
	case 0b10:
		disp = parseUintFromBytes([]byte{bytes[3], bytes[2]})
		cmd.dest = sourceAddrTable[rm] + " + " + strconv.FormatUint(disp, 10)
		data := parseUintFromBytes(bytes[4:5])
		dataLen = 3
		if word == 0b1 {
			isWord = true
			if s == 0b0 {
				data = parseUintFromBytes([]byte{bytes[5], bytes[4]})
				dataLen = 4
			}
		} else {
			isByte = true
		}
		cmd.source = strconv.FormatUint(data, 10)
	}
	if mod != 0b11 {
		if word == 0b0 {
			isByte = true
		}
		if isWord {
			cmd.dest = "word [" + cmd.dest + "]"
		} else if isByte {
			cmd.dest = "byte [" + cmd.dest + "]"
		} else {
			cmd.dest = "[" + cmd.dest + "]"
		}
	}
	fmt.Printf("immediate to reg cmd='%s', word=%b, reg=%b len=%d\n", cmd.Str(), word, rm, dataLen)
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

func parseImmediateToAcc(b []byte, cmdName string) (command, int) {
	word := b[0] & 0b1
	data := b[1:2]
	if word == 0b1 {
		data = []byte{b[2], b[1]}
	}
	cmd := command{
		cmd:    cmdName,
		dest:   regTable[word],
		source: strconv.FormatUint(parseUintFromBytes(data), 10),
	}
	return cmd, len(data)
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
	// parsing [00101000 11100000 00101101 11101000 00000011 00101100] bytes
	// reg to reg cmd='sub al, ah', word=0, reg=100
	//
	// parsing [00101101 11101000 00000011 00101100 11100010 00101100] bytes
	// reg to reg cmd='mov ax, bp', word=1, reg=101
	fmt.Printf("reg to reg cmd='%s', word=%b, reg=%b\n", cmd.Str(), word, reg)
	return cmd
}

func parseCommand(bytes []byte) (command, []byte) {

	fmt.Printf("parsing %08b bytes\n", bytes[:6])

	opcode := bytes[0] >> 2
	// mov rm to/from register
	mod := bytes[1] >> 6
	word := bytes[0] & 1
	direction := (bytes[0] >> 1) & 1
	rm := bytes[1] & 0b111
	reg := (bytes[1] >> 3) & 0b111

	val, ok := jumpTable[bytes[0]]

	switch {
	case bytes[0]>>4 == 0b1011:
		{
			// immediate to register case
			cmd, datalen := parseImmediateToReg(bytes[0]>>4, bytes)
			return cmd, bytes[datalen+1:]
		}
	case bytes[0]>>2 == 0b100000:
		{
			cmd, datalen := parseImmediateAddToReg(bytes)
			return cmd, bytes[datalen+2:]
		}
	case bytes[0]>>1 == 0b0000010:
		{
			//immediate add to accumulator
			cmd, dataLen := parseImmediateToAcc(bytes, "add")
			return cmd, bytes[dataLen+1:]
		}
	case bytes[0]>>1 == 0b0010110:
		{
			//immediate sub to accumulator
			cmd, dataLen := parseImmediateToAcc(bytes, "sub")
			return cmd, bytes[dataLen+1:]
		}
	case bytes[0]>>1 == 0b0011110:
		{
			//immediate sub to accumulator
			cmd, dataLen := parseImmediateToAcc(bytes, "cmp")
			return cmd, bytes[dataLen+1:]
		}
	case ok:
		{
			cmd := parseJump(val, bytes[1])
			return cmd, bytes[2:]
		}
	case mod == 0b11:
		{
			// reg to reg
			cmd := parseRegToReg(opcode, word, direction, rm, reg)
			return cmd, bytes[2:]
		}

	case mod == 0b00:
		{
			cmd := parseRegToMem(opcode, word, direction, rm, reg)
			return cmd, bytes[2:]
		}
	case mod == 0b01:
		{
			// to reg from mem + 8 bit displacement
			cmd := parseRegToMemAndDisp(opcode, word, direction, rm, reg, bytes[2:3])
			return cmd, bytes[3:]
		}
	case mod == 0b10:
		{
			// to reg from mem + 16 bit displacement
			cmd := parseRegToMemAndDisp(opcode, word, direction, rm, reg, bytes[2:4])
			return cmd, bytes[4:]
		}
	}

	// if mod == 0b11 {
	// 	// reg to reg
	// 	cmd := parseRegToReg(opcode, word, direction, rm, reg)
	// 	return cmd, bytes[2:]
	// }
	// if mod == 0b00 {
	// 	cmd := parseRegToMem(opcode, word, direction, rm, reg)
	// 	return cmd, bytes[2:]
	// }
	// if mod == 0b01 {
	// 	// to reg from mem + 8 bit displacement
	// 	cmd := parseRegToMemAndDisp(opcode, word, direction, rm, reg, bytes[2:3])
	// 	return cmd, bytes[3:]
	// }
	// if mod == 0b10 {
	// 	// to reg from mem + 16 bit displacement
	// 	cmd := parseRegToMemAndDisp(opcode, word, direction, rm, reg, bytes[2:4])
	// 	return cmd, bytes[4:]
	// }
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
