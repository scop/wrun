from urllib.request import urlopen
import xml.etree.ElementTree as ET

ATOM_XMLNS = "http://www.w3.org/2005/Atom"


def latest_atom_version(url: str) -> str:
    with urlopen(url) as f:
        data = ET.parse(f)
    # For GitHub releases, title might differ from tag, look up from id
    id = data.find(f"{{{ATOM_XMLNS}}}entry/{{{ATOM_XMLNS}}}id").text
    # Sloppy, expected e.g. tag:github.com,2008:Repository/6731432/v0.10.0
    tag = id.rpartition("/")[2]
    return tag


def latest_rss2_version(url: str) -> str:
    with urlopen(url) as f:
        data = ET.parse(f)
    version = data.find("channel/item/title").text
    return version
