import { silent, type Log } from '../debug/log';

export interface CameraButton {
  /** Shows the locked or the free icon. */
  show(locked: boolean): void;
}

/** The two icons, as image URLs. */
export interface CameraIcons {
  locked: string;
  free: string;
}

/** The camera icon in the top-right corner (ADR 0004): clicking it does what Y does. */
export function createCameraButton(host: HTMLElement, icons: CameraIcons, onClick: () => void, log: Log = silent): CameraButton {
  const button = document.createElement('button');
  button.type = 'button';
  button.className = 'camera-button';
  const image = document.createElement('img');
  image.alt = '';
  button.appendChild(image);
  // a click mustn't leave the focus on it: keys belong to the controls
  button.addEventListener('mousedown', (event) => event.preventDefault());
  button.addEventListener('click', (event) => {
    event.stopPropagation();
    log('camera icon clicked');
    onClick();
  });
  host.appendChild(button);

  return {
    show(locked) {
      image.src = locked ? icons.locked : icons.free;
      button.title = locked ? 'Camera locked on you (Y)' : 'Free camera (Y)';
      button.setAttribute('aria-label', button.title);
      button.setAttribute('aria-pressed', String(locked));
    },
  };
}
