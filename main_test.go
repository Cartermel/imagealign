package main

import (
	"testing"

	"gocv.io/x/gocv"
	"gocv.io/x/gocv/contrib"
)

// tests aligning a skewed image, since alignment is not 100% accurate we will use image hashing to
// determine a correctness threshold
func TestImageAlignNoMask(t *testing.T) {
	reference := gocv.IMRead("./img/lenna.jpg", gocv.IMReadAnyColor)
	defer reference.Close()

	input := gocv.IMRead("./img/lenna_skewed.jpg", gocv.IMReadAnyColor)
	defer input.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	align(reference, &input, mask, 0.7)

	// use image hashing from contrib to easily compare images
	hasher := contrib.AverageHash{}
	referenceHash := gocv.NewMat()
	inputHash := gocv.NewMat()
	defer referenceHash.Close()
	defer inputHash.Close()

	hasher.Compute(reference, &referenceHash)
	hasher.Compute(input, &inputHash)
	distance := hasher.Compare(referenceHash, inputHash)

	// aligning isn't 100% effective, leave a small threshold
	if distance > 5 {
		t.Errorf("Expected lenna.jpg and lenna_skewed.jpg to match with a hamming distance <= 5 after aligning. Got %f\n", distance)
	}

	input2 := gocv.IMRead("./img/lenna_giga_skewed.jpg", gocv.IMReadAnyColor)
	mask2 := gocv.NewMat()
	defer mask2.Close()

	align(reference, &input2, mask2, 1)

	hasher.Compute(input2, &inputHash)
	distance = hasher.Compare(referenceHash, inputHash)

	if distance > 5 {
		t.Errorf("Expected lenna.jpg and lenna_giga_skewed.jpg to match with a hamming distance <= 5 after aligning. Got %f\n", distance)
	}
}

// tests alignment against the same input image with and without a mask
func TestImageAlignWithMask(t *testing.T) {
	reference := gocv.IMRead("./img/ca-t2.png", gocv.IMReadAnyColor)
	defer reference.Close()

	input := gocv.IMRead("./img/ca-t2-input.jpg", gocv.IMReadAnyColor)
	defer input.Close()

	referenceMask := gocv.NewMat()
	defer referenceMask.Close()

	emptyMask := gocv.NewMat()
	defer emptyMask.Close()

	align(reference, &input, emptyMask, 0.7)

	input2 := gocv.IMRead("./img/ca-t2-input.jpg", gocv.IMReadAnyColor)
	defer input.Close()

	inputMask2 := gocv.IMRead("./img/ca-t2-input-mask.jpg", gocv.IMReadGrayScale)
	defer inputMask2.Close()

	align(reference, &input2, inputMask2, 0.7)

	hasher := contrib.AverageHash{}
	inputHash := gocv.NewMat()
	input2Hash := gocv.NewMat()
	defer inputHash.Close()
	defer input2Hash.Close()

	hasher.Compute(input, &inputHash)
	hasher.Compute(input2, &input2Hash)

	distance := hasher.Compare(input2Hash, inputHash)

	if distance > 5 {
		t.Errorf("Expected input with mask to be roughly the same as normal alignment. Got %f hamming distance\n", distance)
	}
}
