# Build report.tex to PDF with pdflatex, keeping all output in build/
$pdf_mode = 1;
$out_dir = 'build';
@default_files = ('report.tex');

# Take turns with other builds: wait for the lock, held until latexmk exits
use Fcntl qw(:flock);
mkdir $out_dir;
open(our $build_lock, '>', "$out_dir/.build.lock") or die "cannot open build lock: $!\n";
flock($build_lock, LOCK_EX) or die "cannot take build lock: $!\n";

# Render the D2 diagrams whose source changed before every build
system('make', '--no-print-directory', 'diagrams') == 0 or die "diagram render failed\n";
