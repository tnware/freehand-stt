// Pure illustration timeline. Preview text never becomes delivered text until
// a phrase is finalized; this does not perform native recording or insertion.
export function demoTiming(realtime) {
  const recordingMS = realtime ? 2400 : 1600;
  const processingMS = realtime ? 150 : 650;
  const insertionMS = realtime ? 180 : 500;
  return { recordingMS, processingMS, insertionMS,
    cycleMS: recordingMS + processingMS + insertionMS + 1200 };
}

export function demoFrame(sentences, realtime, elapsed) {
  const { recordingMS, processingMS, insertionMS, cycleMS } = demoTiming(realtime);
  const sentenceIndex = Math.min(sentences.length - 1, Math.floor(elapsed / cycleMS));
  const localMS = elapsed - sentenceIndex * cycleMS;
  const sentence = sentences[sentenceIndex];
  const insertionStart = recordingMS + processingMS;
  const finalized = localMS >= recordingMS - 200;
  const phase = realtime
    ? sentenceIndex < sentences.length - 1 || localMS < recordingMS
      ? 'recording' : localMS < insertionStart ? 'processing' : 'done'
    : localMS < recordingMS ? 'recording' : localMS < insertionStart
      ? 'processing' : localMS < insertionStart + insertionMS ? 'inserting' : 'done';
  const length = realtime ? (finalized ? sentence.length : 0)
    : Math.max(0, Math.min(sentence.length, Math.floor((localMS - insertionStart) / insertionMS * sentence.length)));
  const words = sentence.split(' ');
  const count = Math.min(words.length, Math.floor(Math.max(0, localMS - 200) / (recordingMS - 450) * words.length));
  return {
    phase,
    text: [...sentences.slice(0, sentenceIndex), sentence.slice(0, length)].filter(Boolean).join('\n\n'),
    preview: words.slice(0, count).join(' ') || 'Listening…',
    status: { recording: realtime ? (finalized ? 'Completed phrase typed · Still recording' : 'Recording · Preview only') : 'Recording',
      processing: realtime ? 'Finalizing' : 'Transcribing', inserting: 'Inserting text', done: 'Text inserted' }[phase],
  };
}
