package managedruntime

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// All standard Whisper GGML model files in the pinned upstream repository.
// Sizes and SHA256 are the official Hugging Face API's LFS metadata:
// https://huggingface.co/api/models/ggerganov/whisper.cpp/revision/5359861c739e955e79d9a303bcbc70fb988958b1?blobs=true
// Core ML encoder archives are companion assets, not server model files.
// The pinned whisper.cpp server reads language, architecture and quantization
// from each GGML model; these variants use the same completed-STT launch recipe.
// Catalog construction performs no network, model download or process work.
const whisperModelRepository = "ggerganov/whisper.cpp"
const whisperModelRevision = "5359861c739e955e79d9a303bcbc70fb988958b1"

var whisperProvider = ggmlProvider{
	id: WhisperCPP, name: "whisper.cpp", platformRecipe: hostCPURecipe(WhisperCPP),
	backend: compatibility.WhisperCPP, profile: modelprofile.Generic, role: compatibility.Transcription,
	models: []Model{
		// Keep the original entries first and Base as the sole recommendation.
		{ID: "base", Name: "Whisper Base", Description: "Multilingual completed transcription.", Recommended: true},
		{ID: "small", Name: "Whisper Small", Description: "Multilingual completed transcription; more memory and time than Base."},
		{ID: "medium", Name: "Whisper Medium", Description: "Multilingual completed transcription; substantial memory and processing time."},
		{ID: "tiny", Name: "Whisper Tiny", Description: "Multilingual completed transcription."},
		{ID: "tiny-q5_1", Name: "Whisper Tiny Q5_1", Description: "Multilingual completed transcription. Q5_1 quantization."},
		{ID: "tiny-q8_0", Name: "Whisper Tiny Q8_0", Description: "Multilingual completed transcription. Q8_0 quantization."},
		{ID: "tiny.en", Name: "Whisper Tiny.en", Description: "English-only completed transcription."},
		{ID: "tiny.en-q5_1", Name: "Whisper Tiny.en Q5_1", Description: "English-only completed transcription. Q5_1 quantization."},
		{ID: "tiny.en-q8_0", Name: "Whisper Tiny.en Q8_0", Description: "English-only completed transcription. Q8_0 quantization."},
		{ID: "base-q5_1", Name: "Whisper Base Q5_1", Description: "Multilingual completed transcription. Q5_1 quantization."},
		{ID: "base-q8_0", Name: "Whisper Base Q8_0", Description: "Multilingual completed transcription. Q8_0 quantization."},
		{ID: "base.en", Name: "Whisper Base.en", Description: "English-only completed transcription."},
		{ID: "base.en-q5_1", Name: "Whisper Base.en Q5_1", Description: "English-only completed transcription. Q5_1 quantization."},
		{ID: "base.en-q8_0", Name: "Whisper Base.en Q8_0", Description: "English-only completed transcription. Q8_0 quantization."},
		{ID: "small-q5_1", Name: "Whisper Small Q5_1", Description: "Multilingual completed transcription. Q5_1 quantization."},
		{ID: "small-q8_0", Name: "Whisper Small Q8_0", Description: "Multilingual completed transcription. Q8_0 quantization."},
		{ID: "small.en", Name: "Whisper Small.en", Description: "English-only completed transcription."},
		{ID: "small.en-q5_1", Name: "Whisper Small.en Q5_1", Description: "English-only completed transcription. Q5_1 quantization."},
		{ID: "small.en-q8_0", Name: "Whisper Small.en Q8_0", Description: "English-only completed transcription. Q8_0 quantization."},
		{ID: "medium-q5_0", Name: "Whisper Medium Q5_0", Description: "Multilingual completed transcription. Q5_0 quantization."},
		{ID: "medium-q8_0", Name: "Whisper Medium Q8_0", Description: "Multilingual completed transcription. Q8_0 quantization."},
		{ID: "medium.en", Name: "Whisper Medium.en", Description: "English-only completed transcription."},
		{ID: "medium.en-q5_0", Name: "Whisper Medium.en Q5_0", Description: "English-only completed transcription. Q5_0 quantization."},
		{ID: "medium.en-q8_0", Name: "Whisper Medium.en Q8_0", Description: "English-only completed transcription. Q8_0 quantization."},
		{ID: "large-v1", Name: "Whisper Large v1", Description: "Multilingual completed transcription."},
		{ID: "large-v2", Name: "Whisper Large v2", Description: "Multilingual completed transcription."},
		{ID: "large-v2-q5_0", Name: "Whisper Large v2 Q5_0", Description: "Multilingual completed transcription. Q5_0 quantization."},
		{ID: "large-v2-q8_0", Name: "Whisper Large v2 Q8_0", Description: "Multilingual completed transcription. Q8_0 quantization."},
		{ID: "large-v3", Name: "Whisper Large v3", Description: "Multilingual completed transcription."},
		{ID: "large-v3-q5_0", Name: "Whisper Large v3 Q5_0", Description: "Multilingual completed transcription. Q5_0 quantization."},
		{ID: "large-v3-turbo", Name: "Whisper Large v3 Turbo", Description: "Multilingual completed transcription."},
		{ID: "large-v3-turbo-q5_0", Name: "Whisper Large v3 Turbo Q5_0", Description: "Multilingual completed transcription. Q5_0 quantization."},
		{ID: "large-v3-turbo-q8_0", Name: "Whisper Large v3 Turbo Q8_0", Description: "Multilingual completed transcription. Q8_0 quantization."},
	},
	specs: map[string]modelSpec{
		"base":                {whisperModelRepository, whisperModelRevision, "ggml-base.bin", "60ed5bc3dd14eea856493d334349b405782ddcaf0028d4b5df4088345fba2efe", 147951465},
		"small":               {whisperModelRepository, whisperModelRevision, "ggml-small.bin", "1be3a9b2063867b937e64e2ec7483364a79917e157fa98c5d94b5c1fffea987b", 487601967},
		"medium":              {whisperModelRepository, whisperModelRevision, "ggml-medium.bin", "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208", 1533763059},
		"tiny":                {whisperModelRepository, whisperModelRevision, "ggml-tiny.bin", "be07e048e1e599ad46341c8d2a135645097a538221678b7acdd1b1919c6e1b21", 77691713},
		"tiny-q5_1":           {whisperModelRepository, whisperModelRevision, "ggml-tiny-q5_1.bin", "818710568da3ca15689e31a743197b520007872ff9576237bda97bd1b469c3d7", 32152673},
		"tiny-q8_0":           {whisperModelRepository, whisperModelRevision, "ggml-tiny-q8_0.bin", "c2085835d3f50733e2ff6e4b41ae8a2b8d8110461e18821b09a15c40c42d1cca", 43537433},
		"tiny.en":             {whisperModelRepository, whisperModelRevision, "ggml-tiny.en.bin", "921e4cf8686fdd993dcd081a5da5b6c365bfde1162e72b08d75ac75289920b1f", 77704715},
		"tiny.en-q5_1":        {whisperModelRepository, whisperModelRevision, "ggml-tiny.en-q5_1.bin", "c77c5766f1cef09b6b7d47f21b546cbddd4157886b3b5d6d4f709e91e66c7c2b", 32166155},
		"tiny.en-q8_0":        {whisperModelRepository, whisperModelRevision, "ggml-tiny.en-q8_0.bin", "5bc2b3860aa151a4c6e7bb095e1fcce7cf12c7b020ca08dcec0c6d018bb7dd94", 43550795},
		"base-q5_1":           {whisperModelRepository, whisperModelRevision, "ggml-base-q5_1.bin", "422f1ae452ade6f30a004d7e5c6a43195e4433bc370bf23fac9cc591f01a8898", 59707625},
		"base-q8_0":           {whisperModelRepository, whisperModelRevision, "ggml-base-q8_0.bin", "c577b9a86e7e048a0b7eada054f4dd79a56bbfa911fbdacf900ac5b567cbb7d9", 81768585},
		"base.en":             {whisperModelRepository, whisperModelRevision, "ggml-base.en.bin", "a03779c86df3323075f5e796cb2ce5029f00ec8869eee3fdfb897afe36c6d002", 147964211},
		"base.en-q5_1":        {whisperModelRepository, whisperModelRevision, "ggml-base.en-q5_1.bin", "4baf70dd0d7c4247ba2b81fafd9c01005ac77c2f9ef064e00dcf195d0e2fdd2f", 59721011},
		"base.en-q8_0":        {whisperModelRepository, whisperModelRevision, "ggml-base.en-q8_0.bin", "a4d4a0768075e13cfd7e19df3ae2dbc4a68d37d36a7dad45e8410c9a34f8c87e", 81781811},
		"small-q5_1":          {whisperModelRepository, whisperModelRevision, "ggml-small-q5_1.bin", "ae85e4a935d7a567bd102fe55afc16bb595bdb618e11b2fc7591bc08120411bb", 190085487},
		"small-q8_0":          {whisperModelRepository, whisperModelRevision, "ggml-small-q8_0.bin", "49c8fb02b65e6049d5fa6c04f81f53b867b5ec9540406812c643f177317f779f", 264464607},
		"small.en":            {whisperModelRepository, whisperModelRevision, "ggml-small.en.bin", "c6138d6d58ecc8322097e0f987c32f1be8bb0a18532a3f88f734d1bbf9c41e5d", 487614201},
		"small.en-q5_1":       {whisperModelRepository, whisperModelRevision, "ggml-small.en-q5_1.bin", "bfdff4894dcb76bbf647d56263ea2a96645423f1669176f4844a1bf8e478ad30", 190098681},
		"small.en-q8_0":       {whisperModelRepository, whisperModelRevision, "ggml-small.en-q8_0.bin", "67a179f608ea6114bd3fdb9060e762b588a3fb3bd00c4387971be4d177958067", 264477561},
		"medium-q5_0":         {whisperModelRepository, whisperModelRevision, "ggml-medium-q5_0.bin", "19fea4b380c3a618ec4723c3eef2eb785ffba0d0538cf43f8f235e7b3b34220f", 539212467},
		"medium-q8_0":         {whisperModelRepository, whisperModelRevision, "ggml-medium-q8_0.bin", "42a1ffcbe4167d224232443396968db4d02d4e8e87e213d3ee2e03095dea6502", 823369779},
		"medium.en":           {whisperModelRepository, whisperModelRevision, "ggml-medium.en.bin", "cc37e93478338ec7700281a7ac30a10128929eb8f427dda2e865faa8f6da4356", 1533774781},
		"medium.en-q5_0":      {whisperModelRepository, whisperModelRevision, "ggml-medium.en-q5_0.bin", "76733e26ad8fe1c7a5bf7531a9d41917b2adc0f20f2e4f5531688a8c6cd88eb0", 539225533},
		"medium.en-q8_0":      {whisperModelRepository, whisperModelRevision, "ggml-medium.en-q8_0.bin", "43fa2cd084de5a04399a896a9a7a786064e221365c01700cea4666005218f11c", 823382461},
		"large-v1":            {whisperModelRepository, whisperModelRevision, "ggml-large-v1.bin", "7d99f41a10525d0206bddadd86760181fa920438b6b33237e3118ff6c83bb53d", 3094623691},
		"large-v2":            {whisperModelRepository, whisperModelRevision, "ggml-large-v2.bin", "9a423fe4d40c82774b6af34115b8b935f34152246eb19e80e376071d3f999487", 3094623691},
		"large-v2-q5_0":       {whisperModelRepository, whisperModelRevision, "ggml-large-v2-q5_0.bin", "3a214837221e4530dbc1fe8d734f302af393eb30bd0ed046042ebf4baf70f6f2", 1080732091},
		"large-v2-q8_0":       {whisperModelRepository, whisperModelRevision, "ggml-large-v2-q8_0.bin", "fef54e6d898246a65c8285bfa83bd1807e27fadf54d5d4e81754c47634737e8c", 1656129691},
		"large-v3":            {whisperModelRepository, whisperModelRevision, "ggml-large-v3.bin", "64d182b440b98d5203c4f9bd541544d84c605196c4f7b845dfa11fb23594d1e2", 3095033483},
		"large-v3-q5_0":       {whisperModelRepository, whisperModelRevision, "ggml-large-v3-q5_0.bin", "d75795ecff3f83b5faa89d1900604ad8c780abd5739fae406de19f23ecd98ad1", 1081140203},
		"large-v3-turbo":      {whisperModelRepository, whisperModelRevision, "ggml-large-v3-turbo.bin", "1fc70f774d38eb169993ac391eea357ef47c88757ef72ee5943879b7e8e2bc69", 1624555275},
		"large-v3-turbo-q5_0": {whisperModelRepository, whisperModelRevision, "ggml-large-v3-turbo-q5_0.bin", "394221709cd5ad1f40c46e6031ca61bce88931e6e088c188294c6d5a55ffa7e2", 574041195},
		"large-v3-turbo-q8_0": {whisperModelRepository, whisperModelRevision, "ggml-large-v3-turbo-q8_0.bin", "317eb69c11673c9de1e1f0d459b253999804ec71ac4c23c17ecf5fbe24e259a1", 874188075},
	},
}
