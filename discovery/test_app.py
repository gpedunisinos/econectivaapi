import unittest
import spacy
from app import Entry, Request, extract


class DiscoveryTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.rules = spacy.blank("pt")

    def test_names_accents_and_word_boundaries(self):
        request = Request(source_id=1, name="Pessoa", text="Estudei na UNISINOS. Ensino na São José. Não é UnisinosX.", entries=[Entry(id=1, names=["Pessoa"]), Entry(id=2, names=["Unisinos"]), Entry(id=3, names=["Sao Jose"])])
        result = extract(request, self.rules)
        self.assertEqual({2, 3}, {r["target_id"] for r in result["references"]})
        self.assertIn("UNISINOS", result["references"][0]["evidence"])
        request.text = "A UnisinosX não é uma correspondência por palavra inteira."
        self.assertEqual([], extract(request, self.rules)["references"])

    def test_mention_is_not_an_alias_of_a_person(self):
        request = Request(source_id=1, name="Bruna", text="Estudei na Universidade do Vale do Rio dos Sinos (Unisinos).", entries=[Entry(id=2, names=["Universidade do Vale do Rio dos Sinos"])])
        self.assertEqual([], extract(request, self.rules)["alias_candidates"])

    def test_self_definition_only_proposes_pending_candidates(self):
        request = Request(source_id=2, name="Universidade do Vale do Rio dos Sinos", text="Universidade do Vale do Rio dos Sinos (Unisinos).", entries=[Entry(id=2, names=["Universidade do Vale do Rio dos Sinos"])])
        self.assertEqual("Unisinos", extract(request, self.rules)["alias_candidates"][0]["name"])
        request.name = "Universidade do Vale do Rio dos Sinos (Unisinos)"
        self.assertIn("Unisinos", [a["name"] for a in extract(request, self.rules)["alias_candidates"]])

    def test_ambiguous_names_retain_both_candidates(self):
        request = Request(source_id=1, name="Projeto", text="Colaboração com o Instituto ABC.", entries=[Entry(id=2, names=["Instituto ABC"]), Entry(id=3, names=["Instituto ABC"])])
        self.assertEqual({2, 3}, {r["target_id"] for r in extract(request, self.rules)["references"]})

    def test_trained_portuguese_model_runs_on_cpu(self):
        model = spacy.load("pt_core_news_sm", exclude=["parser", "lemmatizer", "morphologizer", "attribute_ruler"])
        request = Request(source_id=1, name="Projeto", text="A Universidade de São Paulo participou do encontro em Porto Alegre.", entries=[])
        result = extract(request, model)
        self.assertIsInstance(result["unresolved"], list)
        self.assertTrue(result["unresolved"])


if __name__ == "__main__":
    unittest.main()
