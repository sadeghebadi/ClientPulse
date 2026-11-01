package pdf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PDFValidationResult contains extracted metrics from automated PDF inspection.
type PDFValidationResult struct {
	PageCount     int
	ExtractableText string
	HasPersianText  bool
	FoundSections []string
	MissingSections []string
}

// ValidatePDF automatedly analyzes a generated PDF using pdftotext and pdfinfo.
func ValidatePDF(ctx context.Context, pdfBytes []byte, requiredPhrases []string, expectPersian bool) (*PDFValidationResult, error) {
	tempDir, err := os.MkdirTemp("", "pdf_validate_*")
	if err != nil {
		return nil, fmt.Errorf("failed creating temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	pdfPath := filepath.Join(tempDir, "sample.pdf")
	if err := os.WriteFile(pdfPath, pdfBytes, 0600); err != nil {
		return nil, fmt.Errorf("failed saving pdf: %w", err)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 1. pdfinfo for metadata & page count
	infoCmd := exec.CommandContext(cmdCtx, "pdfinfo", pdfPath)
	infoOut, err := infoCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed running pdfinfo: %w", err)
	}

	pageCount := 0
	for _, line := range strings.Split(string(infoOut), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Pages:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pageCount, _ = strconv.Atoi(parts[1])
			}
		}
	}
	if pageCount < 1 {
		return nil, fmt.Errorf("pdf has invalid page count: %d", pageCount)
	}

	// 2. pdftotext for selectable UTF-8 text extraction
	textCmd := exec.CommandContext(cmdCtx, "pdftotext", "-enc", "UTF-8", pdfPath, "-")
	var stdoutBuf, stderrBuf bytes.Buffer
	textCmd.Stdout = &stdoutBuf
	textCmd.Stderr = &stderrBuf

	if err := textCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed extracting pdf text with pdftotext: %w, stderr: %s", err, stderrBuf.String())
	}

	extracted := stdoutBuf.String()
	if len(strings.TrimSpace(extracted)) == 0 {
		return nil, fmt.Errorf("pdf contains no selectable text (blank or scanned-only)")
	}

	res := &PDFValidationResult{
		PageCount:       pageCount,
		ExtractableText: extracted,
		FoundSections:   make([]string, 0),
		MissingSections: make([]string, 0),
	}

	// Check required phrases
	for _, phrase := range requiredPhrases {
		if strings.Contains(extracted, phrase) {
			res.FoundSections = append(res.FoundSections, phrase)
		} else {
			res.MissingSections = append(res.MissingSections, phrase)
		}
	}

	if len(res.MissingSections) > 0 {
		return res, fmt.Errorf("pdf text is missing required sections/phrases: %v", res.MissingSections)
	}

	// Check Persian character presence if expected
	if expectPersian {
		hasPersian := false
		for _, r := range extracted {
			// Arabic / Persian Unicode blocks: 0600–06FF, 0750–077F, FB50–FDFF, FE70–FEFF
			if (r >= 0x0600 && r <= 0x06FF) || (r >= 0xFB50 && r <= 0xFDFF) {
				hasPersian = true
				break
			}
		}
		res.HasPersianText = hasPersian
		if !hasPersian {
			return res, fmt.Errorf("expected Persian text in PDF, but none was detected in extracted text")
		}
	}

	return res, nil
}

