import 'package:flutter/material.dart';

import 'app.dart';
import 'ui/notes_page.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final app = await AppServices.create();
  runApp(TypewriterNotes(app: app));
  app.syncNow(quiet: true); // a quiet try when the app opens: it does nothing if the computer is not there
}

class TypewriterNotes extends StatelessWidget {
  const TypewriterNotes({super.key, required this.app});
  final AppServices app;

  // the same blue as Typewriter and its icon (frontend/src/app.css): brighter on dark, deeper on light so white text stays easy to read
  static const accent = Color(0xFF3D8BF6);
  static const accentLight = Color(0xFF1F6FE0);

  ThemeData _theme(Brightness b) {
    final dark = b == Brightness.dark;
    final scheme = ColorScheme.fromSeed(seedColor: accent, brightness: b).copyWith(
      surface: dark ? const Color(0xFF0C1118) : const Color(0xFFF3F6FA),
      primary: dark ? accent : accentLight,
    );
    return ThemeData(
      colorScheme: scheme,
      useMaterial3: true,
      scaffoldBackgroundColor: scheme.surface,
      appBarTheme: AppBarTheme(backgroundColor: scheme.surface, surfaceTintColor: Colors.transparent),
      cardTheme: CardThemeData(color: dark ? const Color(0xFF121A24) : Colors.white, elevation: 0, shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12), side: BorderSide(color: dark ? const Color(0xFF23303E) : const Color(0xFFDDE5EF)))),
      inputDecorationTheme: InputDecorationTheme(border: OutlineInputBorder(borderRadius: BorderRadius.circular(10))),
    );
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
        title: 'Typewriter Notes',
        debugShowCheckedModeBanner: false,
        theme: _theme(Brightness.light),
        darkTheme: _theme(Brightness.dark),
        home: NotesPage(app: app),
      );
}
