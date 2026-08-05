import { Link } from 'react-router-dom';
import { motion } from 'motion/react';
import './Home.css';

const MotionLink = motion.create(Link);

const FLOATERS = [
  { word: 'Octopus', x: 6, y: 12, delay: 0 },
  { word: 'Moonwalk', x: 82, y: 18, delay: 1.2 },
  { word: 'Firefighter', x: 14, y: 66, delay: 0.6 },
  { word: 'Jellyfish', x: 76, y: 70, delay: 1.8 },
  { word: 'Air Guitar', x: 44, y: 6, delay: 2.4 },
];

const STEPS = [
  {
    title: 'Create a room',
    body: 'Pick a name and an avatar, then share your room code.',
  },
  {
    title: 'Act it out',
    body: 'One player mimes a secret word on camera. No talking!',
  },
  {
    title: 'Guess & score',
    body: 'Teammates type guesses as fast as they can. Beat the clock.',
  },
];

function Home() {
  return (
    <div className="home">
      <div className="home-floaters" aria-hidden="true">
        {FLOATERS.map(({ word, x, y, delay }) => (
          <motion.span
            key={word}
            className="home-floater"
            style={{ left: `${x}%`, top: `${y}%` }}
            animate={{ y: [0, -14, 0] }}
            transition={{ duration: 5, repeat: Infinity, ease: 'easeInOut', delay }}
          >
            {word}
          </motion.span>
        ))}
      </div>

      <main className="home-main">
        <motion.div
          className="home-hero"
          initial={{ opacity: 0, y: 24, scale: 0.96 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ type: 'spring', stiffness: 120, damping: 16 }}
        >
          <h1 className="home-logo">Charrade</h1>
          <p className="home-tagline">Act it out. Guess it right. Beat the clock.</p>
          <MotionLink
            className="home-cta"
            to="/start"
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.96 }}
            transition={{ type: 'spring', stiffness: 300, damping: 18 }}
          >
            Play now
          </MotionLink>
        </motion.div>

        <motion.ol
          className="home-steps"
          initial="hidden"
          animate="show"
          variants={{
            hidden: {},
            show: { transition: { staggerChildren: 0.12, delayChildren: 0.3 } },
          }}
        >
          {STEPS.map((step, index) => (
            <motion.li
              key={step.title}
              className="home-step"
              variants={{
                hidden: { opacity: 0, y: 16 },
                show: {
                  opacity: 1,
                  y: 0,
                  transition: { type: 'spring', stiffness: 120, damping: 16 },
                },
              }}
            >
              <span className="home-step-num" aria-hidden="true">
                {index + 1}
              </span>
              <h2 className="home-step-title">{step.title}</h2>
              <p className="home-step-body">{step.body}</p>
            </motion.li>
          ))}
        </motion.ol>
      </main>
    </div>
  );
}

export default Home;
