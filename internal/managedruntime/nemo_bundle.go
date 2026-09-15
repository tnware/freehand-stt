package managedruntime

import (
	"context"
	"os"
	"path/filepath"
)

// All bundle pins are from NVIDIA/NeMo-Speech.cpp v0.1.0 models/index.json.
// The model manager acquires Magpie, the 32 MiB tokenizer tar prefix and its
// codec companion. We verify every executable input after its extraction.
var nemoCodecSpec = modelSpec{"nvidia/nemo-nano-codec-22khz-1.89kbps-21.5fps", "fc00890b604aa2de298d2641ffc6c5f6caf8c4d7", "nemo_nano_codec_22khz_1.89kbps_21.5fps.decoder.f16.gguf", "cc86d36d821a27cdc1d4ef600a3e2b0dabe76e88fcc2a8652d9543134c07ef2d", 78823104}
var nemoTokenizerArchive = modelSpec{"nvidia/magpie_tts_multilingual_357m", "452ef560f972c38d5fc16476259aac9456453547", "magpie_tts_multilingual_357m.nemo", "2b930b399933dd384b17ec16a5f1ebf921e2017b760c4fd608c11077a68793fc", 33554432}

var nemoTokenizerMembers = []struct {
	name, sha256 string
	size         int64
}{
	{"05adc40366e149b69319acd4b28a4919_pt_br_prondict-v1.0.dict", "6492abc404db16bbad83dd9d7f9a60eb617699f0a3ff54b3be6184e14507b745", 3580284},
	{"339da71c54b046f98cbcf38ef6d4ff67_hindi_phoneme_merged_phoneme_dict.dict", "978a7aa5a0e3334b13c19015cc61f3ff1b96ce6ac7a31c482e5c22a61f19586c", 7320312},
	{"41913ebaa70342058574da74293b7630_magpie_tts_multilingual_357m.nemo.speakers.json", "36cdcf01ebc0afb506660ac71f2d0a211236374ad5c872d7d8d985a3f9f6ccaf", 68},
	{"61d57a4ccb064d1e8e3d9f871c571932_heteronyms-052722", "b701909aedf753172eff223950f8859cd4b9b4c80199cf0a6e9ac4a307c8f8ec", 1606},
	{"74b3e832fe914a569fcd42b51d06b2f1_ipa_dict_nv23.05.txt", "252e8eaf60dfd891520759913dc8534393eebfa8d57192365f76f9229f294e9e", 5419},
	{"7dbc31751f224f2486090d59dc95b9f7_es_ES_nv230301.dict", "94c9a25bb359f733cd863c887d0112e7ae19a744b0333ef64b6a88e576428669", 2230910},
	{"c5e4ec2af5a14ce294f4b9edcc936535_de_nv230119.heteronym", "771cc585a574fd35bd14f4ce6108edf1e0e512a8dbc810db7de4c1165ba9d0ef", 44566},
	{"cf01ab5c48c84f3282ef7888263361e5_de_nv230119.dict", "5c7bbf3346ebd6dc57769b5cc805124215cb1bcc3b8174c4254e9e928c5d6094", 4313907},
	{"dc7d60d6b15a4651b21c9ca2932b62c6_ipa_cmudict-0.7b_nv23.01.txt", "dd0927fffc89e8539ea0a26ccbc164a908f4b3de9613d924c90afa5300e00f72", 3093097},
	{"model_config.yaml", "fda01948a40a04316b26553b53546cbd2103952b8d0e2813176ea33852bb96a2", 7496},
}

func nemoModelSize(id string) int64 {
	size := modelSpecs[id].size
	if id == "magpie-tts" {
		size += nemoCodecSpec.size + nemoTokenizerArchive.size
	}
	return size
}

func nemoTokenizerDirectory(root string) string {
	return filepath.Join(nemoTokenizerArchive.directory(root), "tokenizer")
}

func verifyNeMoModel(ctx context.Context, root, id string) error {
	spec := modelSpecs[id]
	if err := verifyFile(ctx, spec.path(root), spec.size, spec.sha256); err != nil {
		return err
	}
	if id != "magpie-tts" {
		return nil
	}
	if err := verifyFile(ctx, nemoCodecSpec.path(root), nemoCodecSpec.size, nemoCodecSpec.sha256); err != nil {
		return err
	}
	dir := nemoTokenizerDirectory(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) != len(nemoTokenizerMembers) {
		return errIntegrity
	}
	for _, member := range nemoTokenizerMembers {
		if err := verifyFile(ctx, filepath.Join(dir, member.name), member.size, member.sha256); err != nil {
			return err
		}
	}
	return nil
}

func nemoDownloadSpecs() []modelSpec {
	specs := make([]modelSpec, 0, len(modelSpecs)+2)
	for _, spec := range modelSpecs {
		specs = append(specs, spec)
	}
	return append(specs, nemoCodecSpec, nemoTokenizerArchive)
}

func nemoModelAcquiredBytes(root, id string) AcquisitionProgress {
	progress := nemoAcquiredBytes(root, modelSpecs[id])
	if id != "magpie-tts" {
		return progress
	}
	progress.TotalBytes = nemoModelSize(id)
	progress.Bytes += nemoAcquiredBytes(root, nemoCodecSpec).Bytes
	tokenizer := nemoAcquiredBytes(root, nemoTokenizerArchive).Bytes
	// The CLI deletes the tar prefix once its verified tokenizer is extracted.
	// File metadata only: extraction completion is not integrity acceptance.
	if tokenizer == 0 {
		complete := safeRoot(nemoTokenizerDirectory(root)) == nil
		for _, member := range nemoTokenizerMembers {
			if !complete {
				break
			}
			info, err := os.Lstat(filepath.Join(nemoTokenizerDirectory(root), member.name))
			if err != nil || !info.Mode().IsRegular() || info.Size() != member.size || isReparse(filepath.Join(nemoTokenizerDirectory(root), member.name)) {
				complete = false
				break
			}
		}
		if complete {
			tokenizer = nemoTokenizerArchive.size
		}
	}
	progress.Bytes += tokenizer
	return boundedAcquisition(progress)
}
