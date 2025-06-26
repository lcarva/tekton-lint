package main

import (
	"fmt"
	"reflect"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func main() {
	taskRef := &v1.TaskRef{}
	t := reflect.TypeOf(*taskRef)
	
	fmt.Printf("TaskRef fields:\n")
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fmt.Printf("  %s: %s\n", field.Name, field.Type)
	}
	
	fmt.Printf("\nResolverRef fields:\n")
	resolverRef := &v1.ResolverRef{}
	rt := reflect.TypeOf(*resolverRef)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fmt.Printf("  %s: %s\n", field.Name, field.Type)
	}
} 