// Compile the pinned libfvad translation units as one private unit so the Go
// toolchain can build the library without a separate CMake installation.
#include "upstream/src/signal_processing/division_operations.c"
#include "upstream/src/signal_processing/energy.c"
#include "upstream/src/signal_processing/get_scaling_square.c"
#include "upstream/src/signal_processing/resample_48khz.c"
#include "upstream/src/signal_processing/resample_by_2_internal.c"
#include "upstream/src/signal_processing/resample_fractional.c"
#include "upstream/src/signal_processing/spl_inl.c"
#include "upstream/src/vad/vad_core.c"
#include "upstream/src/vad/vad_filterbank.c"
#include "upstream/src/vad/vad_gmm.c"
#include "upstream/src/vad/vad_sp.c"
#include "upstream/src/fvad.c"
