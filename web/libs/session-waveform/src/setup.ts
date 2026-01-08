import { library } from '@fortawesome/fontawesome-svg-core';

import { faWaveSquare } from '@fortawesome/free-solid-svg-icons/faWaveSquare';
import { faMicrochip } from '@fortawesome/free-solid-svg-icons/faMicrochip';
import { faHeart } from '@fortawesome/free-solid-svg-icons/faHeart';
import { faTrash } from '@fortawesome/free-solid-svg-icons/faTrash';
import { faThumbtack } from '@fortawesome/free-solid-svg-icons/faThumbtack';
import { faArrowLeft } from '@fortawesome/free-solid-svg-icons/faArrowLeft';
import { faPlus } from '@fortawesome/free-solid-svg-icons/faPlus';
import { faMinus } from '@fortawesome/free-solid-svg-icons/faMinus';
import { faMagnifyingGlassChart } from '@fortawesome/free-solid-svg-icons/faMagnifyingGlassChart';
import { faArrowUpRightDots } from '@fortawesome/free-solid-svg-icons/faArrowUpRightDots';
import { faTimes } from '@fortawesome/free-solid-svg-icons/faTimes';
import { faPlay } from '@fortawesome/free-solid-svg-icons/faPlay';
import { faPause } from '@fortawesome/free-solid-svg-icons/faPause';
import { faClock } from '@fortawesome/free-regular-svg-icons/faClock';
import { faArrowsLeftRightToLine } from '@fortawesome/free-solid-svg-icons/faArrowsLeftRightToLine';
import { faDownload } from '@fortawesome/free-solid-svg-icons/faDownload';
import { faMusic } from '@fortawesome/free-solid-svg-icons/faMusic';
import { faScissors } from '@fortawesome/free-solid-svg-icons/faScissors';
import { faCheckCircle } from '@fortawesome/free-solid-svg-icons/faCheckCircle';
import { faExclamationCircle } from '@fortawesome/free-solid-svg-icons/faExclamationCircle';
import { faExclamationTriangle } from '@fortawesome/free-solid-svg-icons/faExclamationTriangle';
import { faInfoCircle } from '@fortawesome/free-solid-svg-icons/faInfoCircle';
import { faStop } from '@fortawesome/free-solid-svg-icons/faStop';
import { faCheck } from '@fortawesome/free-solid-svg-icons/faCheck';
import { faRedo } from '@fortawesome/free-solid-svg-icons/faRedo';

export const setup = () => {
  library.add(
    faTimes,
    faPlay,
    faPause,
    faStop,
    faCheck,
    faMicrochip,
    faWaveSquare,
    faTrash,
    faHeart,
    faThumbtack,
    faArrowLeft,
    faPlus,
    faMinus,
    faMagnifyingGlassChart,
    faArrowUpRightDots,
    faArrowsLeftRightToLine,
    faClock,
    faDownload,
    faMusic,
    faScissors,
    faCheckCircle,
    faExclamationCircle,
    faExclamationTriangle,
    faInfoCircle,
    faRedo
  );
};
