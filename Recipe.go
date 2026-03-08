//go:build gobake
package bake_recipe

import (
	"fmt"
	"github.com/fezcode/gobake"
	"runtime"
)

func Run(bake *gobake.Engine) error {
	if err := bake.LoadRecipeInfo("recipe.piml"); err != nil {
		return err
	}

	bake.Task("build", "Builds the binary", func(ctx *gobake.Context) error {
		ctx.Log("Building %s v%s...", bake.Info.Name, bake.Info.Version)

		err := ctx.Mkdir("build")
		if err != nil {
			return err
		}

		ldflags := fmt.Sprintf("-X main.Version=%s", bake.Info.Version)
		output := "build/" + bake.Info.Name
		if runtime.GOOS == "windows" {
			output += ".exe"
		}

		return ctx.Run("go", "build", "-ldflags", ldflags, "-o", output, ".")
	})

	bake.Task("clean", "Removes build artifacts", func(ctx *gobake.Context) error {
		return ctx.Remove("build")
	})

	return nil
}
