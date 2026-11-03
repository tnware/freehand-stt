import test from 'node:test';
import assert from 'node:assert/strict';
import { demoFrame, demoTiming } from '../src/lib/dictation-demo.mjs';
const sentences = ['First completed phrase.', 'Second completed phrase.'];

test('live previews stay out of the document until the whole phrase is final', () => {
  assert.equal(demoFrame(sentences, true, 1700).text, '');
  assert.notEqual(demoFrame(sentences, true, 1700).preview, 'Listening…');
  const final = demoFrame(sentences, true, 2200);
  assert.equal(final.text, sentences[0]);
  assert.equal(final.preview, sentences[0]);
  assert.equal(final.phase, 'recording');
});
test('live typing preserves prior phrases while the next preview develops', () => {
  const { cycleMS } = demoTiming(true);
  assert.equal(demoFrame(sentences, true, cycleMS + 1000).text, sentences[0]);
  assert.equal(demoFrame(sentences, true, cycleMS + 2200).text, sentences.join('\n\n'));
  assert.equal(demoFrame(sentences, true, cycleMS + 2200).phase, 'recording');
  assert.equal(demoFrame(sentences, true, cycleMS + 2400).phase, 'processing');
  assert.equal(demoFrame(sentences, true, cycleMS + 2550).phase, 'done');
});
test('completed mode waits for Stop and processing before insertion', () => {
  assert.equal(demoFrame(sentences, false, 1600).text, '');
  assert.equal(demoFrame(sentences, false, 2249).text, '');
  assert.equal(demoFrame(sentences, false, 2500).phase, 'inserting');
  assert.equal(demoFrame(sentences, false, 2750).text, sentences[0]);
});
test('reduced-motion final frame is complete and replay starts clean', () => {
  for (const realtime of [false, true]) {
    const duration = demoTiming(realtime).cycleMS * sentences.length + 1800;
    assert.equal(demoFrame(sentences, realtime, duration - 1).text, sentences.join('\n\n'));
    assert.equal(demoFrame(sentences, realtime, duration - 1).phase, 'done');
    assert.equal(demoFrame(sentences, realtime, 0).text, '');
  }
});
