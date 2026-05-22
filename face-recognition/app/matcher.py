"""SIMS API face match client with Redis cache and fallback."""

import logging
import json
import time
from typing import Optional

import httpx
import numpy as np

logger = logging.getLogger(__name__)


def cosine_similarity(a: np.ndarray, b: np.ndarray) -> float:
    """Compute cosine similarity between two L2-normalized vectors.

    Both vectors should already be unit-norm.  The result is clipped
    to [-1.0, 1.0] to protect against float drift.
    """
    dot = float(np.dot(a, b))
    return max(-1.0, min(1.0, dot))


class FaceMatcher:
    """Face identity matcher backed by the SIMS REST API and a Redis cache.

    Flow
    ----
    1. Query the SIMS API with the face embedding.
    2. On success → cache the result, return it with ``from_cache=False``.
    3. On API failure (connection error, timeout) → fall back to scanning
       the Redis cache for the closest cached embedding.
    4. If cache also misses → return ``None``.
    """

    def __init__(self, config):
        self.config = config
        self._redis = None
        self._init_redis()

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _init_redis(self):
        try:
            import redis as redis_mod

            rc = self.config.redis if hasattr(self.config, 'redis') else None
            host = rc.host if rc else "localhost"
            port = rc.port if rc else 6379
            db_idx = rc.db if rc else 0
            timeout = rc.socket_timeout if rc else 2.0
            self._redis = redis_mod.Redis(
                host=host,
                port=port,
                db=db_idx,
                decode_responses=False,
                socket_connect_timeout=timeout,
                socket_timeout=timeout,
            )
            self._redis.ping()
            logger.info("Redis connected for face-match cache (%s:%s/%d)", host, port, db_idx)
        except Exception:
            logger.warning("Redis unavailable, face-match cache disabled", exc_info=True)
            self._redis = None

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def match(self, embedding: np.ndarray) -> Optional[dict]:
        """Return match result or ``None``.

        Returns
        -------
        dict or None
            ``{"student_id": str, "name": str, "confidence": float, "from_cache": bool}``
        """
        # 1. Primary path: SIMS API
        try:
            result = self._call_sims_api(embedding)
            if result is not None:
                self._cache_result(result["student_id"], embedding, result["name"])
                result["from_cache"] = False
                return result
        except Exception as exc:
            logger.warning("SIMS API call failed, falling back to cache", error=str(exc))

        # 2. Fallback: scan cached embeddings
        if self.config.fallback_to_cache:
            cached = self._search_cache(embedding)
            if cached is not None:
                cached["from_cache"] = True
                return cached

        return None

    # ------------------------------------------------------------------
    # SIMS API
    # ------------------------------------------------------------------

    def _call_sims_api(self, embedding: np.ndarray) -> Optional[dict]:
        payload = {"embedding": embedding.tolist()}
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.config.get_auth_token()}",
        }
        response = httpx.post(
            self.config.sims_api_url,
            json=payload,
            headers=headers,
            timeout=self.config.sims_api_timeout,
        )
        response.raise_for_status()
        data = response.json()

        match_info = data.get("match")
        # Support both nested (Go API) and flat (legacy SpringBoot) response formats
        if isinstance(match_info, dict):
            conf = match_info.get("confidence", 0)
            if conf >= self.config.match_threshold:
                return {
                    "student_id": match_info["student_id"],
                    "name": match_info.get("name", ""),
                    "confidence": conf,
                }
        elif match_info and data.get("confidence", 0) >= self.config.match_threshold:
            return {
                "student_id": data["student_id"],
                "name": data.get("name", ""),
                "confidence": data["confidence"],
            }
        return None

    # ------------------------------------------------------------------
    # Redis cache helpers
    # ------------------------------------------------------------------

    def _cache_result(self, student_id: str, embedding: np.ndarray, name: str):
        if self._redis is None:
            return
        try:
            value = {
                "embedding": embedding.tolist(),
                "name": name,
                "timestamp": time.time(),
            }
            self._redis.setex(
                f"face:match:{student_id}",
                self.config.cache_ttl,
                json.dumps(value),
            )
        except Exception:
            logger.warning("Failed to cache match result in Redis", exc_info=True)

    def _search_cache(self, query_emb: np.ndarray) -> Optional[dict]:
        """Scan all cached embeddings and return the best match above threshold."""
        if self._redis is None:
            return None
        try:
            # Use SCAN instead of KEYS — non-blocking, cursor-based
            keys = []
            cursor = 0
            while True:
                cursor, batch = self._redis.scan(cursor=cursor, match="face:match:*", count=100)
                keys.extend(batch)
                if cursor == 0:
                    break

            best_sim = 0.0
            best_result = None

            # Pipeline all GET requests
            pipe = self._redis.pipeline()
            for key in keys:
                pipe.get(key)
            results = pipe.execute()

            for key, raw in zip(keys, results):
                if raw is None:
                    continue
                data = json.loads(raw)
                cached_emb = np.array(data["embedding"], dtype=np.float32)
                sim = cosine_similarity(query_emb, cached_emb)
                if sim > best_sim:
                    best_sim = sim
                    best_result = {
                        "student_id": key.decode().split(":", 2)[2],
                        "name": data["name"],
                        "confidence": sim,
                    }

            if best_result is not None and best_sim >= self.config.match_threshold:
                return best_result
        except Exception:
            logger.warning("Cache search failed", exc_info=True)

        return None
