package main

import "google.golang.org/protobuf/compiler/protogen"

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		protos := parseProtocols(gen.Request.GetParameter())
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			if err := generateFile(gen, f, protos); err != nil {
				return err
			}
		}
		generateAggregates(gen)
		return nil
	})
}
