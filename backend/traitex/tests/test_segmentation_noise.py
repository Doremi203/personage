from app.services.segmentation.noise import is_noise_message


def test_empty_text_with_no_media_is_noise():
    assert is_noise_message("", None) is True
    assert is_noise_message(None, None) is True


def test_sticker_without_caption_is_noise():
    assert is_noise_message("", "sticker") is True


def test_photo_without_caption_is_not_noise():
    # A photo without caption is still meaningful content (visual evidence).
    assert is_noise_message("", "photo") is False


def test_video_without_caption_is_not_noise():
    assert is_noise_message("", "video") is False


def test_short_emoji_only_is_noise():
    assert is_noise_message("👍", None) is True
    assert is_noise_message("🙏🙏", None) is True
    assert is_noise_message("...", None) is True


def test_known_acknowledgement_phrases_are_noise():
    assert is_noise_message("ок", None) is True
    assert is_noise_message("OK", None) is True
    assert is_noise_message("спасибо", None) is True
    assert is_noise_message("Thanks", None) is True


def test_substantive_short_text_is_not_noise():
    assert is_noise_message("в 19:30 у входа", None) is False
    assert is_noise_message("Привет!", None) is False


def test_emoji_with_caption_is_not_noise():
    assert is_noise_message("👍 договорились", None) is False


def test_text_with_media_is_never_treated_as_noise_phrase():
    # If there's media, the message carries content even if the caption is "ok".
    assert is_noise_message("ок", "photo") is False
