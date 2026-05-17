"""Low-light frame enhancement using CLAHE during night hours."""

from datetime import datetime

import cv2
import numpy as np


class NightModeEnhancer:
    """Enhance low-light frames with CLAHE on the L-channel of LAB colour space.

    Enhancement is only applied when:
    * ``config.enabled`` is ``True``, AND
    * The current local hour falls within the configured night window
      (e.g. 22:00 – 06:00).
    """

    def __init__(self, config):
        self.config = config
        self._clahe = cv2.createCLAHE(
            clipLimit=config.clahe_clip_limit,
            tileGridSize=(8, 8),
        )

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def is_night(self) -> bool:
        """Return ``True`` if the current hour is in the night window.

        The night window spans ``[start_hour, end_hour)`` with wrap-around:
        e.g. start_hour=22, end_hour=6 means 22:00 – 05:59.
        """
        h = datetime.now().hour
        return h >= self.config.start_hour or h < self.config.end_hour

    def enhance(self, frame: np.ndarray) -> np.ndarray:
        """Return an enhanced frame (BGR) or the original unchanged."""
        if not self.config.enabled:
            return frame
        if not self.is_night():
            return frame

        # Convert BGR → LAB, apply CLAHE to L channel, convert back
        lab = cv2.cvtColor(frame, cv2.COLOR_BGR2LAB)
        l_ch, a_ch, b_ch = cv2.split(lab)
        l_enhanced = self._clahe.apply(l_ch)
        lab_enhanced = cv2.merge([l_enhanced, a_ch, b_ch])
        return cv2.cvtColor(lab_enhanced, cv2.COLOR_LAB2BGR)
