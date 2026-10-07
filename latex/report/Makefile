# Build the D2 diagrams and the LaTeX final report.
#
# Requires:
#   * d2     -- `go install github.com/d2lang/d2@latest`; found on PATH or
#               in Go's bin folder
#   * TeX Live with pdflatex, latexmk and pgfplots
#
# latexmk reads .latexmkrc, renders changed diagrams, reruns pdflatex until
# the cross-references settle, and writes everything to build/. LaTeX
# Workshop runs the same latexmk build whenever report.tex changes; a lock in
# .latexmkrc makes builds that start at the same time run one after another.
#
# Usage:
#   make            # render diagrams (if changed) and build build/report.pdf
#   make diagrams   # only (re)render the D2 diagrams
#   make clean      # remove LaTeX intermediate files, keep the PDF
#   make distclean  # also remove build/ and the rendered figures

D2      ?= $(shell command -v d2 || echo $$(go env GOPATH)/bin/d2)
D2FLAGS := --pad 25
LATEXMK := latexmk -interaction=nonstopmode -halt-on-error -synctex=1
DOC     := report

# every .d2 source in the repo's diagrams/ folder; each sets its own layout
DIAGRAM_DIR := ../../diagrams
SOURCES     := $(wildcard $(DIAGRAM_DIR)/*.d2)
FIGS        := $(SOURCES:$(DIAGRAM_DIR)/%.d2=figures/%.pdf)

.PHONY: all diagrams clean distclean
all: diagrams
	$(LATEXMK)
	@echo "==> build/$(DOC).pdf ready"

diagrams: $(FIGS)

figures/%.pdf: $(DIAGRAM_DIR)/%.d2 | figures
	$(D2) $(D2FLAGS) $< $@

figures:
	mkdir -p figures

clean:
	latexmk -c

distclean:
	rm -rf build $(FIGS)
