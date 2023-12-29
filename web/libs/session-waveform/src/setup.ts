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
import { faStopwatch } from '@fortawesome/free-solid-svg-icons/faStopwatch';
import { faTimes } from '@fortawesome/free-solid-svg-icons/faTimes';
import { faPlay } from '@fortawesome/free-solid-svg-icons/faPlay';
import { faPause } from '@fortawesome/free-solid-svg-icons/faPause';

export const setup = () => {
  library.add(
    faTimes,
    faPlay,
    faPause,
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
    faStopwatch
  );
};
