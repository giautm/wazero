package wasi_sql_test

import (
	"context"
	"fmt"
	"log"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_sql"
)

// Example demonstrates how to instantiate the wasi:sql module.
func Example() {
	ctx := context.Background()

	// Create a new runtime
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Instantiate the wasi:sql module
	wasi_sql.MustInstantiate(ctx, r)

	// The module is now available for WebAssembly modules to import
	fmt.Println("wasi:sql module instantiated successfully")

	// Output:
	// wasi:sql module instantiated successfully
}

// ExampleBuilder demonstrates how to use the Builder pattern for more control.
func ExampleBuilder() {
	ctx := context.Background()

	// Create a new runtime
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Create a builder
	builder := wasi_sql.NewBuilder(r)

	// Compile the module (optional, for ahead-of-time compilation)
	compiled, err := builder.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer compiled.Close(ctx)

	// Instantiate the compiled module
	closer, err := builder.Instantiate(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close(ctx)

	fmt.Println("wasi:sql module compiled and instantiated")

	// Output:
	// wasi:sql module compiled and instantiated
}

// ExampleFunctionExporter demonstrates how to export wasi:sql functions to a custom module.
func ExampleFunctionExporter() {
	ctx := context.Background()

	// Create a new runtime
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Create a custom module name
	customModuleName := "my-custom-sql-module"

	// Use FunctionExporter to add wasi:sql functions to your custom module
	builder := r.NewHostModuleBuilder(customModuleName)
	exporter := wasi_sql.NewFunctionExporter()
	exporter.ExportFunctions(builder)

	// Instantiate the custom module
	_, err := builder.Instantiate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Custom module %q instantiated with wasi:sql functions\n", customModuleName)

	// Output:
	// Custom module "my-custom-sql-module" instantiated with wasi:sql functions
}
