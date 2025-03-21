FROM jekyll/builder:3

COPY Gemfile Gemfile.lock ./

RUN gem install bundler -v 2.2.28 && bundle install