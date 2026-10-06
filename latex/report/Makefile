# Build the D2 diagrams and the LaTeX final report.
#
# Requires:
#   * d2     -- `go install github.com/d2lang/d2@latest`; if it isn't on
#               PATH, run `make D2=$(go env GOPATH)/bin/d2`
#   * TeX Live with pdflatex + pgfplots
#
# Usage:
#   make            # render diagrams (if changed) and build the PDF
#   make diagrams   # only (re)render the D2 diagrams
#   make clean      # remove LaTeX intermediate files
#   make distclean  # also remove the PDF and the rendered figures

D2       ?= d2
D2FLAGS  := --pad 25
PDFLATEX := pdflatex -interaction=nonstopmode -halt-on-error -synctex=1
DOC      := report

# every .d2 source in the repo's diagrams/ folder; each sets its own layout
DIAGRAM_DIR := ../../diagrams
SOURCES     := $(wildcard $(DIAGRAM_DIR)/*.d2)
FIGS        := $(SOURCES:$(DIAGRAM_DIR)/%.d2=figures/%.png)

.PHONY: all diagrams clean distclean
all: $(DOC).pdf
	@echo "==> $(DOC).pdf ready"

diagrams: $(FIGS)

figures/%.png: $(DIAGRAM_DIR)/%.d2 | figures
	$(D2) $(D2FLAGS) $< $@

figures:
	mkdir -p figures

# run twice for the table of contents and pgfplots
$(DOC).pdf: $(DOC).tex $(FIGS)
	$(PDFLATEX) $(DOC).tex
	$(PDFLATEX) $(DOC).tex
	@rm -f $(DOC).aux $(DOC).log $(DOC).out $(DOC).toc

clean:
	rm -f $(DOC).aux $(DOC).log $(DOC).out $(DOC).toc

distclean: clean
	rm -f $(DOC).pdf $(DOC).synctex.gz $(FIGS)
