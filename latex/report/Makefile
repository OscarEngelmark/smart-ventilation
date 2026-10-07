# Build the D2 diagrams and the LaTeX final report.
#
# Requires:
#   * d2     -- `go install github.com/d2lang/d2@latest`; if it isn't on
#               PATH, run `make D2=$(go env GOPATH)/bin/d2`
#   * TeX Live with pdflatex, latexmk and pgfplots
#
# latexmk reads .latexmkrc, reruns pdflatex until the cross-references
# settle, and writes everything to build/. Saving report.tex in VS Code runs
# the same latexmk build through LaTeX Workshop.
#
# Usage:
#   make            # render diagrams (if changed) and build build/report.pdf
#   make diagrams   # only (re)render the D2 diagrams
#   make clean      # remove LaTeX intermediate files, keep the PDF
#   make distclean  # also remove build/ and the rendered figures

D2      ?= d2
D2FLAGS := --pad 25
LATEXMK := latexmk -interaction=nonstopmode -halt-on-error -synctex=1
DOC     := report

# every .d2 source in the repo's diagrams/ folder; each sets its own layout
DIAGRAM_DIR := ../../diagrams
SOURCES     := $(wildcard $(DIAGRAM_DIR)/*.d2)
FIGS        := $(SOURCES:$(DIAGRAM_DIR)/%.d2=figures/%.png)

.PHONY: all diagrams clean distclean
all: diagrams
	$(LATEXMK)
	@echo "==> build/$(DOC).pdf ready"

diagrams: $(FIGS)

figures/%.png: $(DIAGRAM_DIR)/%.d2 | figures
	$(D2) $(D2FLAGS) $< $@

figures:
	mkdir -p figures

clean:
	latexmk -c

distclean:
	rm -rf build $(FIGS)
