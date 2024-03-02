package cc

import (
	"github.com/kelindar/gocc/internal/asm"
	"github.com/kelindar/gocc/internal/config"
)

type Disassembler struct {
	arch         *config.Arch
	disassembler []string
}

func NewDisassembler(arch *config.Arch) (*Disassembler, error) {
	var (
		dasm    []string
		objdump string
		err     error
	)

	if arch.UseGoObjdump {
		objdump, err = config.FindGoObjdump()
		dasm = append(dasm, objdump, "tool", "objdump")
	} else {
		objdump, err = config.FindClangObjdump()
		dasm = append(dasm, objdump)
	}
	if err != nil {
		return nil, err
	}

	return &Disassembler{
		arch:         arch,
		disassembler: dasm,
	}, nil
}

// Disassemble disassembles the object file
func (d *Disassembler) Disassemble(assemblyPath, objectPath string) ([]asm.Function, error) {
	// Parse the assembly file
	assembly, err := asm.ParseAssembly(d.arch, assemblyPath)
	if err != nil {
		return nil, err
	}

	disassembler := d.disassembler
	if d.arch.Disassembler != nil {
		disassembler = append(disassembler, d.arch.Disassembler...)
	}
	if !d.arch.UseGoObjdump {
		disassembler = append(disassembler, "-d")
	}
	disassembler = append(disassembler, objectPath)

	// Run the disassembler
	dump, err := runCommand(disassembler[0], disassembler[1:]...)
	if err != nil {
		return nil, err
	}

	// Parse the object dump and map machine code to assembly
	if d.arch.UseGoObjdump {
		err = asm.ParseGoObjectDump(d.arch, dump, assembly)
	} else {
		err = asm.ParseClangObjectDump(d.arch, dump, assembly)
	}
	if err != nil {
		return nil, err
	}

	return assembly, nil
}
