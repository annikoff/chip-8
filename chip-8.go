package main

import (
	"github.com/nsf/termbox-go"
	//"encoding/hex"
	//"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"
)

type Chip8 struct {
	opcode     byte       //двухбайтовый опкод
	memory     [4096]byte //массив памяти
	V          [16]byte   //16 восьмибитных регистров общего назначения и флаг переноса VF
	I          uint16     //адресный регистр
	pc         uint16     //указатель кода
	sp         int        //указатель стека
	stack      [16]uint16 //массив стека
	delayTimer int        //таймер задержки
	soundTimer int        //таймер звука

	screen [64 * 32]bool //массив, представляющий экран
	key    [16]byte      //массив, представляющий клавиатуру
	stop   bool          //переменная для опкода 00FD
	mode   int
}

func (c *Chip8) Init() {
	for i := 0; i < 4096; i++ {
		c.memory[i] = 0x0
	}
	for row := 0; row < 32; row++ {
		for col := 0; col < 64; col++ {
			c.screen[row*64+col] = false
		}
	}
	for i := 0; i < 16; i++ {
		c.V[i] = 0
	}
	for i := 0; i < 16; i++ {
		c.key[i] = 0
	}
	for i := 0; i < 16; i++ {
		c.stack[i] = 0
	}
	c.pc = 0x200 // программа начинается со смещения 0x200
	c.I = 0
	fonts := [86]byte{0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
		0x20, 0x60, 0x20, 0x20, 0x70, // 1
		0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
		0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
		0x90, 0x90, 0xF0, 0x10, 0x10, // 4
		0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
		0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
		0xF0, 0x10, 0x20, 0x40, 0x40, // 7
		0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
		0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
		0xF0, 0x90, 0xF0, 0x90, 0x90, // A
		0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
		0xF0, 0x80, 0x80, 0x80, 0xF0, // C
		0xE0, 0x90, 0x90, 0x90, 0xE0, // D
		0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
		0xF0, 0x80, 0xF0, 0x80, 0x80} // F

	for i, char := range fonts {
		c.memory[i] = char
	}
	c.delayTimer = 0
	c.soundTimer = 0
	c.stop = false
}

func (c *Chip8) Step() {
	var op uint16
	if c.pc >= uint16(len(c.memory)) {
		log.Fatal("The ROM file is too large!")
		return
	}
	op = (uint16(c.memory[c.pc]) << 8) | uint16(c.memory[c.pc+1])
	switch (op & 0xF000) >> 12 {
	case 0x0:
		switch op & 0xFF {
		case 0xE0: // 00E0. Clear the screen.
			for row := 0; row < 32; row++ {
				for col := 0; col < 64; col++ {
					c.screen[row*64+col] = false
				}
			}
		case 0xEE: // 00EE.Returns from a subroutine.
			c.sp--
			c.pc = c.stack[c.sp]
			return
		case 0xFD: // 00FD. Quit the emulator.
			c.stop = true
			log.Print("Quit the emulator")
			return
		case 0xFE: // 00FE. disable extended screen mode *SCHIP*.
			c.mode = 0
			break
		case 0xFF: // 00FF. enable extended screen mode *SCHIP*.
			c.mode = 1
			break
		default:
			log.Fatal("Invalid opcode! 1 ", op)
			return
		}
	case 0x1: // 1NNN. Jumps to address NNN.
		c.pc = op & 0xFFF
	case 0x2: // 2NNN. Calls subroutine at NNN.
		c.stack[c.sp] = c.pc + 2
		c.sp++
		c.pc = op & 0xFFF
	case 0x3: // 3XNN. Skips the next instruction if VX equals NN.
		if c.V[(op&0x0F00)>>8] == byte(op&0xFF) {
			c.pc += 2
		}
	case 0x4: // 4XNN. Skips the next instruction if VX doesn't equal NN.
		if c.V[(op&0x0F00)>>8] != byte(op&0xFF) {
			c.pc += 2
		}
	case 0x5: // 6XNN. Skips the next instruction if VX equals VY.
		if c.V[(op&0x0F00)>>8] == c.V[(op&0x00F0)>>4] {
			c.pc += 2
		}
	case 0x6: // 5XY0. Sets VX to NN.
		c.V[(op&0x0F00)>>8] = byte(op & 0xFF)
	case 0x7: // 7XNN. Adds NN to VX.
		c.V[(op&0x0F00)>>8] += byte(op & 0xFF)
	case 0x8:
		x := (op & 0x0F00) >> 8
		y := (op & 0x00F0) >> 4
		switch op & 0x000F {
		case 0x0: // 8XY0. Sets VX to the value of VY.
			c.V[x] = c.V[y]
		case 0x1: // 8XY1. Sets VX to VX or VY.
			c.V[x] |= c.V[y]
		case 0x2: // 8XY2. Sets VX to VX and VY.
			c.V[x] &= c.V[y]
		case 0x3: // 8XY3. Sets VX to VX xor VY.
			c.V[x] ^= c.V[y]
		case 0x4: // 8XY4. Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't.
			c.V[x] += c.V[y]
		case 0x5: // 8XY5. VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
			c.V[x] -= c.V[y]
		case 0x6: // 8XY6. Shifts VX right by one. VF is set to the value of the least significant bit of VX before the shift.
		case 0x7: // 8XY7. Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
			c.V[x] = c.V[y] - c.V[x]
			if c.V[y] > c.V[x] {
				c.V[0xF] = 1
			} else {
				c.V[0xF] = 0
			}
		case 0xE: // 8XYE. Shifts VX left by one. VF is set to the value of the most significant bit of VX before the shift.
			c.V[0xF] = (c.V[((op&0x0F00)>>8)] >> 7) & 0x01
			c.V[((op & 0x0F00) >> 8)] <<= 1
		}
	case 0x9: // 9XY0. Skips the next instruction if VX doesn't equal VY.
		if c.V[(op&0x0F00)>>8] != c.V[(op&0x00F0)>>4] {
			c.pc += 2
		}
	case 0xA: // ANNN. Sets I to the address NNN.
		c.I = op & 0xFFF
	case 0xB: // BNNN. Jumps to the address NNN plus V0.
		c.pc = (op & 0xFFF) + uint16(c.V[0])
	case 0xC: // CXNN. Sets VX to a random number and NN.
		c.V[(op&0x0F00)>>8] = byte(rand.Intn(int(op&0xFF) + 1))
	case 0xD: // DXYN. Draws a sprite at coordinate (VX, VY) that has a width of 8 pixels and a height of N pixels.
		x := (op & 0x0F00) >> 8
		y := (op & 0x00F0) >> 4
		s := byte(op & 0x000F)
		c.draw(c.V[x], c.V[y], s)
	case 0xE:
		switch op & 0x00FF {
		case 0x9E: // EX9E. Skip next instruction if key VX down.
			if c.key[c.V[((op&0x0F00)>>8)]] == 1 {
				c.pc += 2
			}
			break
		case 0xA1: // EXA1. Skip next instruction if key VX up.
			if c.key[c.V[((op&0x0F00)>>8)]] == 0 {
				c.pc += 2
			}
			break

		default:
			log.Fatal("Invalid opcode!", op)
			return
		}
		break
	case 0xF:
		switch op & 0x00FF {
		case 0x07: // FX07. Sets VX = delayTimer.
			c.V[((op & 0x0F00) >> 8)] = byte(c.delayTimer)
			break
		case 0x0a: // FX0A. Sets VX = key, wait for keypress.
			c.pc -= 2
			for n := 0; n < 16; n++ {
				if c.key[n] == 1 {
					c.V[((op & 0x0F00) >> 8)] = byte(n)
					c.pc += 2
					break
				}
			}
			break
		case 0x15: // FX15. Sets delayTimer = VX.
			c.delayTimer = int(c.V[((op & 0x0F00) >> 8)])
			break
		case 0x18: // FX18. Sets soundTimer = VX.
			c.soundTimer = int(c.V[((op & 0x0F00) >> 8)])
			break
		case 0x1E: // FX1E. Sets I = I + VX; set VF if buffer overflow.
			I := c.I + uint16(c.V[((op&0x0F00)>>8)])
			if I > 0xFFF {
				c.V[0xF] = 1
			} else {
				c.V[0xF] = 0
			}
			break
		case 0x29: // FX29. Point I to 5 byte numeric sprite for value in VX.
			c.I = uint16(c.V[((op&0x0F00)>>8)] * 5)
			break
		case 0x30: // FX30. Point I to 10 byte numeric sprite for value in VX *SCHIP*.
			c.I = uint16(c.V[((op&0x0F00)>>8)]*10 + 80)
			break
		case 0x33: // FX33. Store BCD of VX in [I], [I+1], [I+2].
			n := c.V[((op & 0x0F00) >> 8)]
			c.memory[c.I] = (n - (n % 100)) / 100
			n -= c.memory[c.I] * 100
			c.memory[c.I+1] = (n - (n % 10)) / 10
			n -= c.memory[c.I+1] * 10
			c.memory[c.I+2] = n
			break
		case 0x55: // FX55 - store V0 .. VX in [I] .. [I+X].
			for n := 0; n <= int((op&0x0F00)>>8); n++ {
				c.memory[c.I+1] = c.V[n]
			}
			break
		case 0x65: // FX65 - read V0 .. VX from [I] .. [I+X].
			for n := 0; n <= int((op&0x0F00)>>8); n++ {
				c.V[n] = c.memory[c.I+1]
			}
			break
		case 0x75: // FX75. Save V0...VX (X<8) in the HP48 flags *SCHIP*.
			/*for i:=0; i <= ((op & 0x0F00)>>8); i++ {
				c.hp48Flags[i] = c.V[i]
			}*/
			break
		case 0x85: // FX85. Load V0...VX (X<8) from the HP48 flags *SCHIP*.
			/*for i:=0; i <= ((op & 0x0F00)>>8); i++ {
				c.V[i] = c.hp48Flags[i]
			}*/
			break

		default:
			log.Fatal("Invalid opcode! 2 ", op)
			return
		}
		break
	default:
		log.Fatal("Invalid opcode! 3 ", op)
		return
	}
	c.pc += 2
}

func (c *Chip8) timersDown() {
	if c.delayTimer > 0 {
		c.delayTimer--
	}
	if c.soundTimer > 0 {
		c.soundTimer--
	}
}

func (c *Chip8) draw(x, y, size byte) {
	c.V[0xF] = 0
	if size == 0 {
		size = 16
	}
	for col := 0; col < 8; col++ {
		for row := 0; row < int(size); row++ {
			px := int(x) + col
			py := int(y) + row
			bit := (c.memory[c.I+uint16(row)] & (1 << uint(col))) != 0
			if px < 64 && py < 32 && px >= 0 && py >= 0 {
				src := c.screen[py*64+px]
				dst := (bit != src)
				c.screen[py*64+px] = dst
				if src && !dst {
					c.V[0xF] = 1
				}
			}
		}
	}
}

func (c *Chip8) LoadGame(rom []byte) {
	if len(rom) > len(c.memory)-0x200 {
		log.Fatal("The ROM file is too large!")
		return
	}
	copy(c.memory[0x200:], rom)
}

func main() {

	romname := "games/IBM Logo.c8"

	chip := new(Chip8)
	chip.Init()

	rom, err := ioutil.ReadFile(romname)
	if err != nil {
		log.Fatalf("Error: %s", err)
		return
	}

	chip.LoadGame(rom)

	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputAlt)

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()
	for {
		if chip.stop {
			return
		}
		select {
		case ev := <-eventQueue:
			if ev.Type == termbox.EventKey {
				switch ev.Key {
				case termbox.KeyF1:
					chip.key[1] = 1
					break
				case termbox.KeyF2:
					chip.key[2] = 1
					break
				case termbox.KeyF3:
					chip.key[3] = 1
					break
				case termbox.KeyF4:
					chip.key[0xC] = 1
					break
				case termbox.KeyF5:
					chip.key[4] = 1
					break
				case termbox.KeyF6:
					chip.key[5] = 1
					break
				case termbox.KeyF7:
					chip.key[6] = 1
					break
				case termbox.KeyF8:
					chip.key[0xD] = 1
					break
				case termbox.KeyF9:
					chip.key[7] = 1
					break
				case termbox.KeyF10:
					chip.key[8] = 1
					break
				case termbox.KeyF11:
					chip.key[9] = 1
					break
				case termbox.KeyF12:
					chip.key[0xE] = 1
					break
				case termbox.KeyArrowLeft:
					chip.key[0xA] = 1
					break
				case termbox.KeyArrowRight:
					chip.key[0] = 1
					break
				case termbox.KeyArrowUp:
					chip.key[0xB] = 1
					break
				case termbox.KeyArrowDown:
					chip.key[0xF] = 1
					break
				case termbox.KeyEsc:
					return
					break
				default:
					for i := 0; i < 16; i++ {
						chip.key[i] = 0
					}
				}
			}
			break
		default:
			termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
			chip.timersDown()
			chip.Step()

			color := termbox.ColorDefault
			for row := 0; row < 32; row++ {
				for col := 0; col < 64; col++ {
					if chip.screen[row*64+col] {
						color = termbox.ColorBlue
					} else {
						color = termbox.ColorDefault
					}
					termbox.SetCell(col, row, ' ', termbox.ColorDefault, color)
				}
			}
			termbox.Flush()
			time.Sleep(time.Millisecond * 60)
		}
	}
}
