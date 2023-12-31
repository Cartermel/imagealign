package main

import (
	"fmt"
	"image"
	"os"
	"sort"

	"github.com/jessevdk/go-flags"
	"gocv.io/x/gocv"
)

var args struct {
	Percent       float64 `short:"p" description:"Percentage of 'good matches' to use from opencv feature detection, represented by a float64 value ranging from 0-1" default:"0.7"`
	InputPath     string  `short:"i" long:"input" description:"Input image path." required:"true"`
	ReferencePath string  `short:"r" long:"reference" description:"Reference image path." required:"true"`
	InputMask     *string `short:"m" long:"mask" description:"Mask for feature detection on the input image"`
	OutputPath    string  `short:"o" long:"output" description:"Output path of aligned image." required:"true"`
}

func main() {
	_, err := flags.Parse(&args)

	if args.Percent <= 0 || args.Percent > 1 {
		err = fmt.Errorf("Percentage must be > 0 and < 1.\n")
	}

	if err != nil {
		os.Exit(1)
	}

	input := readMatFromPath(args.InputPath)
	reference := readMatFromPath(args.ReferencePath)

	var inputMask gocv.Mat
	if args.InputMask != nil {
		inputMask = readMatFromPath(*args.InputMask)
	} else {
		inputMask = gocv.NewMat()
	}

	defer input.Close()
	defer reference.Close()
	defer inputMask.Close()

	align(reference, &input, inputMask, args.Percent)
	writeSuccess := gocv.IMWrite(args.OutputPath, input)

	if !writeSuccess {
		fmt.Printf("Failed to write output image to '%s'", args.OutputPath)
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
