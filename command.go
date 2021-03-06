// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gopkg.in/mgo.v2/bson"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("dump")

type dump struct {
	showTxn  bool
	useBSOND bool

	filename string
}

const helpDoc = `
Dump a bson file to json.
`

// Info implements Command.
func (c *dump) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "bsondump",
		Args:    "filename",
		Purpose: "Dump a bson file to json",
		Doc:     helpDoc,
	}
}

func (c *dump) IsSuperCommand() bool {
	return false
}

func (c *dump) AllowInterspersedFlags() bool {
	return true
}

// SetFlags implements Command.
func (c *dump) SetFlags(f *gnuflag.FlagSet) {
	f.BoolVar(&c.showTxn, "txn", false, "Include txn-revno and txn-queue")
	f.BoolVar(&c.useBSOND, "d", false, "Use bson.D rather than bson.M")
}

// Init implements Command.
func (c *dump) Init(args []string) error {
	if len(args) == 0 {
		return errors.Errorf("missing filename")
	}
	c.filename, args = args[0], args[1:]
	return cmd.CheckEmpty(args)
}

// Run implements Command.
func (c *dump) Run(ctx *cmd.Context) error {

	fileContent, err := ioutil.ReadFile(c.filename)
	if err != nil {
		// Try adding ".bson" to the filename
		fileContent, err = ioutil.ReadFile(c.filename + ".bson")
		if err != nil {
			return err
		}
	}

	for {
		var (
			size int32
			doc  []byte
		)
		binary.Read(bytes.NewReader(fileContent), binary.LittleEndian, &size)

		doc, fileContent = fileContent[0:size], fileContent[size:]

		var content interface{}
		if c.useBSOND {
			var value bson.D
			if err := bson.Unmarshal(doc, &value); err != nil {
				return err
			}
			if !c.showTxn {
				var trimmed bson.D
				for _, item := range value {
					switch item.Name {
					case "txn-revno", "txn-queue":
					default:
						trimmed = append(trimmed, item)
					}
				}
				value = trimmed
			}

			content = value
		} else {
			var value bson.M
			if err := bson.Unmarshal(doc, &value); err != nil {
				return err
			}
			if !c.showTxn {
				delete(value, "txn-revno")
				delete(value, "txn-queue")
			}

			content = value
		}

		b, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return err
		}

		ctx.Stdout.Write(b)
		fmt.Fprintln(ctx.Stdout)
		if len(fileContent) == 0 {
			break
		}
	}

	return nil
}
