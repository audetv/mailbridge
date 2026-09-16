package extractor

import "testing"

func TestParseFrom_6F(t *testing.T) {
	cases := []struct{ in, we, wn string }{
		{"A B <a@b.ru>", "a@b.ru", "A B"},
		{"<a@b.ru>", "a@b.ru", ""},
		{"a@b.ru", "a@b.ru", ""},
		{"X <a@b.ru>, second@c.ru", "a@b.ru", "X"},
		{"no-at-bad", "", ""},
		{"=?UTF-8?B?0JjQstCw0L0=?= <a@b.ru>", "a@b.ru", "Иван"},
		// Legacy shape from prod DB — RFC822 без закрывающей скобки:
		{"\u0413\u0443\u0441\u0435\u0432 \u0410\u043b\u0435\u043a\u0441\u0435\u0439 <agusev@gcconsulting.ru", "agusev@gcconsulting.ru", "\u0413\u0443\u0441\u0435\u0432 \u0410\u043b\u0435\u043a\u0441\u0435\u0439"},
		{"postfix@oao-solomon.ru", "postfix@oao-solomon.ru", ""},
	}
	for _, c := range cases {
		email, name := ParseFromHeader(c.in)
		if email != c.we {
			t.Errorf("in=%q email=%q want %q", c.in, email, c.we)
		}
		if name != c.wn {
			t.Errorf("in=%q name=%q want %q", c.in, name, c.wn)
		}
	}
}
