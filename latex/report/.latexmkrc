# Build report.tex to PDF with pdflatex, keeping all output in build/
$pdf_mode = 1;
$out_dir = 'build';
@default_files = ('report.tex');

# Render the D2 diagrams whose source changed before every build
system('make', '--no-print-directory', 'diagrams') == 0 or die "diagram render failed\n";
