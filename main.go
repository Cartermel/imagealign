package main

import (
	"fmt"
	"image"
	"os"
	"sort"

	"github.com/urfave/cli/v2"
	"gocv.io/x/gocv"
)

func main() {
	var inputPath, referencePath, outputPath string
	var percent float64
	var maskPath *string

	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "imagealign",
		Usage:                "Aligns the passed input image with given reference.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "input",
				Aliases:     []string{"i"},
				Required:    true,
				Usage:       "Input image path",
				Destination: &inputPath,
			},
			&cli.StringFlag{
				Name:        "reference",
				Aliases:     []string{"r"},
				Required:    true,
				Usage:       "Reference image path",
				Destination: &referencePath,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Required:    true,
				Usage:       "Output image path",
				Destination: &outputPath,
			},
			&cli.Float64Flag{
				Name:        "percent",
				Aliases:     []string{"p"},
				Usage:       "Percentage of 'good matches' to use from opencv feature detection, represented by a float64 value ranging from 0-1",
				Destination: &percent,
				Value:       0.7,
			},
			&cli.StringFlag{
				Name:        "mask",
				Aliases:     []string{"m"},
				Usage:       "Input image mask path",
				Destination: maskPath,
			},
		},
		Action: func(*cli.Context) error {
			input := readMatFromPath(inputPath)
			reference := readMatFromPath(referencePath)

			var inputMask gocv.Mat
			if maskPath != nil {
				inputMask = readMatFromPath(*maskPath)
			} else {
				inputMask = gocv.NewMat()
			}

			defer input.Close()
			defer reference.Close()
			defer inputMask.Close()

			fmt.Println(percent)

			align(reference, &input, inputMask, percent)
			writeSuccess := gocv.IMWrite(outputPath, input)

			// TODO: handle file extensions, gocv doesnt catch cvexceptions so this panics currently
			if !writeSuccess {
				return fmt.Errorf("Failed to write output image to '%s'", outputPath)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("ERROR: %v", err)
		os.Exit(1)
	}
}

func readMatFromPath(path string) gocv.Mat {
	mat := gocv.IMRead(path, gocv.IMReadAnyColor)

	if mat.Empty() {
		fmt.Printf("Failed to read image from path '%s'\n", path)
		os.Exit(1)
	}

	return mat
}

// Aligns the given input image against the reference using opencv feature detection + homography matrix
// applies alignment directly on the input image
func align(reference gocv.Mat, input *gocv.Mat, inputMask gocv.Mat, percentMatches float64) {
	orb := gocv.NewORB()
	defer orb.Close()

	emptyMask := gocv.NewMat()
	defer emptyMask.Close()

	kpsRef, descRef := orb.DetectAndCompute(reference, emptyMask)
	kpsInput, descInput := orb.DetectAndCompute(*input, inputMask)

	matcher := gocv.NewBFMatcherWithParams(gocv.NormHamming, true)
	defer matcher.Close()

	// match and sort for best matches (slice will be [best matches -> worst matches])
	matches := matcher.Match(descRef, descInput)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Distance < matches[j].Distance
	})

	percentToTake := int(float64(len(matches)) * percentMatches)
	goodMatches := matches[:percentToTake]

	srcPts := gocv.NewMatWithSize(len(goodMatches), 1, gocv.MatTypeCV64FC2)
	dstPts := gocv.NewMatWithSize(len(goodMatches), 1, gocv.MatTypeCV64FC2)
	defer srcPts.Close()
	defer dstPts.Close()

	for i, match := range goodMatches {
		kpRef := kpsRef[match.QueryIdx]
		kpInput := kpsInput[match.TrainIdx]

		srcPts.SetDoubleAt(i, 0, kpRef.X)
		srcPts.SetDoubleAt(i, 1, kpRef.Y)

		dstPts.SetDoubleAt(i, 0, kpInput.X)
		dstPts.SetDoubleAt(i, 1, kpInput.Y)
	}

	homography := gocv.FindHomography(dstPts, &srcPts, gocv.HomograpyMethodRANSAC, 3, &emptyMask, 2000, 0.955)

	size := reference.Size()
	gocv.WarpPerspective(*input, input, homography, image.Point{size[1], size[0]})
}
