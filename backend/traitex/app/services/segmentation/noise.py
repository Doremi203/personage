NOISE_PHRASES: frozenset[str] = frozenset({
    "ок",
    "окей",
    "ok",
    "okay",
    "+",
    "+1",
    "ага",
    "угу",
    "хорошо",
    "thx",
    "thanks",
    "thank you",
    "спс",
    "спасибо",
    "лол",
    "lol",
})


def is_noise_message(text: str | None, media_kind: str | None) -> bool:
    """Return True for low-signal Telegram messages.

    A message is treated as noise when it carries no usable conversational
    payload — pure stickers, reactions, single-emoji acks, or one of a small
    set of culturally-common acknowledgements. Noise messages still appear in
    the rendered transcript when an open segment exists, but they neither
    open a new segment on their own nor reset the silence-window timer.
    """
    stripped = (text or "").strip()

    # No text and either no media at all or only a sticker → reactive noise.
    if not stripped:
        if media_kind is None or media_kind == "sticker":
            return True
        return False

    if media_kind is None:
        if stripped.lower() in NOISE_PHRASES:
            return True
        # Short emoji/punctuation acks: "👍", "👌🙏", "+++", "..."
        if len(stripped) <= 6 and not any(ch.isalnum() for ch in stripped):
            return True
    return False
