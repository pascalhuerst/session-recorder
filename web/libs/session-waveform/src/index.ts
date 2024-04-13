import EmptyScreen from './lib/display/EmptyScreen.vue';
import WaveformEditor from './waveform/WaveformEditor.vue';
import Button from './lib/controls/Button.vue';
import Modal from './lib/disclosure/Modal.vue';
import { useConfirmation } from './lib/disclosure/useConfirmation';
import {
  createPeaksContext,
  providePeaksContext,
  usePeaksContext,
} from './context/usePeaksContext';

export { EmptyScreen, WaveformEditor, Button, Modal, useConfirmation };
export * from './setup';
export { usePeaksContext, createPeaksContext, providePeaksContext };
